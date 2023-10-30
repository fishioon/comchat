package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

type H map[string]interface{}

type server struct {
	rdb  *redis.Client
	chat websocket.Handler
	mux  *http.ServeMux
	log  *zap.Logger
}

func NewServer(log *zap.Logger, redisURL string) (*server, error) {
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379"
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.TODO()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis fail: %w", err)
	}
	s := &server{
		rdb: rdb,
		mux: http.NewServeMux(),
		log: log,
	}
	s.init()
	return s, nil
}

func getString(h map[string]interface{}, k string) string {
	if m, ok := h[k]; ok {
		if s, ok := m.(string); ok {
			return s
		}
	}
	return ""
}

func getNowID() string {
	ts := time.Now().UnixMilli()
	return strconv.FormatInt(ts, 10)
}

type HeartBeat struct {
	URLs []string `json:"urls"`
	SEQs []string `json:"seqs"`
}

func (s *server) Chat(conn *websocket.Conn) {
	// TODO: check auth
	r := conn.Request()
	var (
		groups []redis.XStream
		err    error
		a      = &redis.XReadArgs{
			Count:   100,
			Block:   time.Second * 3,
			Streams: []string{},
		}
		wg           sync.WaitGroup
		activeGroups []string
	)
	groupReadSeq := make(map[string]string)

	getGroupRead := func(gid string) string {
		seq, ok := groupReadSeq[gid]
		if !ok {
			seq = strconv.FormatInt(time.Now().UnixMilli(), 10) + "-0"
			groupReadSeq[gid] = seq
		}
		return seq
	}

	genStreams := func() []string {
		n := len(activeGroups)
		streams := make([]string, n*2)
		for i := 0; i < n; i++ {
			streams[i] = activeGroups[i]
			streams[i+n] = getGroupRead(streams[i])
		}
		return streams
	}

	wg.Add(2)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		buf := make([]byte, 4906)
		heartbeat := &HeartBeat{}
		var (
			n   int
			err error
		)
		for {
			if n, err = conn.Read(buf); err != nil {
				s.log.Error("read websocket failed", zap.Error(err))
				break
			}
			s.log.Sugar().Debugf("recvmsg: %s", string(buf[:n]))
			if err = json.Unmarshal(buf[:n], heartbeat); err != nil {
				s.log.Error("decoded websocket message failed", zap.Error(err))
				break
			}
			activeGroups = heartbeat.URLs
		}
	}(r.Context(), &wg)

	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			a.Streams = genStreams()
			topicCount := len(a.Streams) / 2
			if topicCount == 0 {
				time.Sleep(time.Second)
				continue
			}
			s.log.Debug("listen streams", zap.Strings("streams", a.Streams))
			if groups, err = s.rdb.XRead(ctx, a).Result(); err != nil {
				if err == redis.Nil {
					time.Sleep(time.Millisecond * 10)
					continue
				}
				break
			}
			msgs := []Msg{}
			for _, group := range groups {
				if len(group.Messages) > 0 {
					for _, msg := range group.Messages {
						msg := Msg{
							ID:      getString(msg.Values, "id"),
							GID:     getString(msg.Values, "gid"),
							Content: getString(msg.Values, "content"),
						}
						msgs = append(msgs, msg)
					}
					groupReadSeq[group.Stream] = group.Messages[len(group.Messages)-1].ID
				}
			}
			s.log.Debug("sendmsg", zap.Any("msgs", msgs), zap.Any("groups", groupReadSeq))
			data, _ := json.Marshal(msgs)
			if _, err = conn.Write(data); err != nil {
				break
			}
		}
	}(r.Context(), &wg)
	wg.Wait()
	log.Println("chat end", err)
}

type Session struct {
	// Gids group ids
	uid    string
	conn   *websocket.Conn
	groups map[string]string
}

func (s *Session) Streams() []string {
	n := len(s.groups)
	if n == 0 {
		return []string{}
	}
	ss := make([]string, n*2)
	i := 0
	for k, v := range s.groups {
		ss[i] = k
		ss[i+n] = v
		i++
	}
	return ss
}

type Msg struct {
	ID      string `json:"id"`
	GID     string `json:"gid"`
	Content string `json:"content"`
}

type Group struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

func (s *server) PubMsg(w http.ResponseWriter, r *http.Request) {
	msg := new(Msg)
	if err := s.handleReq(w, r, msg); err != nil {
		return
	}
	// save to db
	// send to redis stream
	ctx := r.Context()
	a := &redis.XAddArgs{
		Stream: msg.GID,
		Values: map[string]interface{}{
			"content": msg.Content,
			"id":      msg.ID,
			"gid":     msg.GID,
		},
	}
	if err := s.rdb.XAdd(ctx, a).Err(); err != nil {
		s.handleError(w, r, err)
		return
	}
	s.respOK(w, r, H{"id": msg.ID})
}

func (s *server) handleGroupDetail(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		URL string `json:"url"`
	}{}
	if err := s.handleReq(w, r, req); err != nil {
		return
	}
	res, err := s.rdb.Get(r.Context(), req.URL).Result()
	if err != nil {
		s.handleError(w, r, err)
		return
	}
	h := H{}
	if err = json.Unmarshal([]byte(res), &h); err != nil {
		s.handleError(w, r, err)
		return
	}
	s.respOK(w, r, h)
}

func redisGroupKey(gid string) string {
	return "g:" + gid
}

type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *server) handleReq(w http.ResponseWriter, r *http.Request, data any) error {
	var (
		body []byte
		err  error
	)
	defer func() {
		if err != nil {
			log.Printf("handle request body failed: %s %s", r.URL.String(), err.Error())
			s.handleError(w, r, err)
		} else {
			log.Printf("handle request body success: %s %s", r.URL.String(), string(body))
		}
	}()
	if body, err = ioutil.ReadAll(r.Body); err != nil {
		return err
	}
	if err = json.Unmarshal(body, data); err != nil {
		return err
	}
	return err
}

func (s *server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	e := &ApiError{500, err.Error()}
	b, _ := json.Marshal(e)
	w.WriteHeader(500)
	w.Write(b)
}

func (s *server) respOK(w http.ResponseWriter, r *http.Request, res interface{}) {
	b, _ := json.Marshal(res)
	w.WriteHeader(200)
	w.Write(b)
}

func (s *server) init() {
	s.mux.HandleFunc("/pub", s.PubMsg)
	s.mux.HandleFunc("/group/detail", s.handleGroupDetail)
	s.mux.Handle("/chat", websocket.Handler(s.Chat))
	s.mux.Handle("/", http.FileServer(http.Dir("./static")))
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
