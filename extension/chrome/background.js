const domain = 'localhost:9981';
const apiURL = `http://${domain}/`;

let webSocket = null;

function connect() {
  if (!webSocket) {
    webSocket = new WebSocket(`ws://${domain}/chat?token=hello`);
    webSocket.onopen = (_event) => {
      console.log('websocket open');
      keepAlive();
    };
    webSocket.onmessage = (event) => {
      console.log(`websocket received message: ${event.data}`);
      const msgs = JSON.parse(event.data);
      sendDanmu(msgs);
    };
    webSocket.onclose = (event) => {
      webSocket = null;
      console.log('websocket connection closed, start reconnect', event);
    };
  }
}

function disconnect() {
  if (webSocket == null) {
    return;
  }
  webSocket.close();
}

async function heartbeat() {
  const tabs = await chrome.tabs.query({});
  const urls = Array.from(new Set(tabs.map(x => fixURL(x.url))));
  webSocket.send(JSON.stringify({ 'urls': urls }));
}

function keepAlive() {
  const keepAliveIntervalId = setInterval(
    async () => {
      if (webSocket) {
        heartbeat();
      } else {
        clearInterval(keepAliveIntervalId);
        connect();
      }
    },
    10 * 1000,
  );
}

async function apiCall(method, msg) {
  const data = {
    method: "POST", // *GET, POST, PUT, DELETE, etc.
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(msg),
  }
  const response = await fetch(apiURL + method, data);
  return response.json();
}

async function sendDanmu(message) {
  const [tab] = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  console.log('sendDanmu===', tab, message);
  const response = await chrome.tabs.sendMessage(tab.id, message);
  console.log(response);
}

function fixURL(s) {
  // remove query
  const i = s.indexOf('?');
  return i === -1 ? s : s.slice(0, i);
}

chrome.tabs.onUpdated.addListener((_tabId, changeInfo, _tab) => {
  if (changeInfo.status === 'complete') {
    heartbeat();
  }
});
chrome.runtime.onMessage.addListener(async (request, sender, sendResponse) => {
  console.log(request, sender);
  const [tab] = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  if (tab) {
    await pubMsg(fixURL(tab.url), request.text);
    sendResponse({ text: "sendok" });
  }
});

// comchat service api

// pubMsg publish message
async function pubMsg(url, text) {
  return apiCall('pub', { gid: url, uid: '', content: text });
}

// getGroupDetail get group detail
async function getGroupDetail(url) {
  return {};
}

connect();
