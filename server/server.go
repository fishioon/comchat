package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/fishioon/comchat/proto"
	"github.com/redis/go-redis/v9"
)

type server struct {
	proto.UnimplementedChatServer
	rdb *redis.Client
}

func NewServer() (*server, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := rdb.Ping(context.TODO()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis fail: %w", err)
	}
	s := &server{
		rdb: rdb,
	}
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

func (s *server) Conn(in *proto.ConnReq, stream proto.Chat_ConnServer) error {
	// 更新 session id
	a := &redis.XReadArgs{
		Count: 100,
		Block: time.Second,
	}
	session := &Session{
		Groups: in.Groups,
	}
	for {
		a.Streams = session.Streams()
		groups, err := s.rdb.XRead(context.TODO(), a).Result()
		if err != nil {
			if err == redis.Nil {
				time.Sleep(time.Millisecond * 10)
				continue
			}
			return err
		}
		msgs := []*proto.Msg{}
		for i, group := range groups {
			if len(group.Messages) > 0 {
				for _, msg := range group.Messages {
					msg := &proto.Msg{
						Id:      getString(msg.Values, "id"),
						Gid:     getString(msg.Values, "gid"),
						Content: getString(msg.Values, "content"),
					}
					msgs = append(msgs, msg)
				}
				session.Groups[i].Seq = group.Messages[len(group.Messages)-1].ID
			}
		}
		if err = stream.Send(&proto.ConnRsp{Msgs: msgs}); err != nil {
			return err
		}
	}
}

type Session struct {
	// Gids group ids
	Groups []*proto.Group
}

func (s *Session) Streams() []string {
	n := len(s.Groups)
	if n == 0 {
		return nil
	}
	ts := time.Now().UnixMilli()
	id := strconv.FormatInt(ts, 10) + "-0"
	ss := make([]string, n*2)
	for i := 0; i < n; i++ {
		g := s.Groups[i]
		ss[i] = g.Id
		if g.Seq == "" {
			g.Seq = id
		}
		ss[i+n] = g.Seq
	}
	return ss
}

func (s *server) getSession(ctx context.Context, sid string) (*Session, error) {
	res, err := s.rdb.Get(ctx, "sess:"+sid).Result()
	if err != nil {
		return nil, err
	}
	sess := &Session{}
	json.Unmarshal([]byte(res), sess)
	return sess, nil
}

func (s *server) PubMsg(ctx context.Context, in *proto.PubMsgReq) (*proto.PubMsgRsp, error) {
	// save to db
	// send msg to redis stream
	err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: in.GetMsg().Gid,
		Values: map[string]interface{}{
			"content": in.GetMsg().GetContent(),
			"id":      in.GetMsg().GetId(),
			"gid":     in.GetMsg().GetGid(),
		},
	}).Err()
	if err != nil {
		return nil, err
	}
	return &proto.PubMsgRsp{}, nil
}
