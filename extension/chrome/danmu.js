// content.js
const danmuContainer = document.createElement('div');
danmuContainer.id = 'danmu-container';
document.body.appendChild(danmuContainer);

// 获取页面背景颜色
const backgroundColor = window.getComputedStyle(document.body).backgroundColor;
// 根据背景颜色选择弹幕文本颜色
const textColor = calculateTextColor(backgroundColor);

// 在 content.js 中
function addDanmu(text) {
  const danmuElement = document.createElement('div');
  danmuElement.className = 'danmu';
  danmuElement.textContent = text;
  danmuElement.style.top = randomTop(10, 80);
  danmuElement.style.color = textColor;
  danmuContainer.appendChild(danmuElement);

  danmuElement.addEventListener('animationend', function () {
    danmuContainer.removeChild(danmuElement);
  });
}

chrome.runtime.onMessage.addListener(function (message) {
  for (const it of message) {
    addDanmu(it.content);
  }
});

function randomTop(from, to) {
  const randomTop = Math.floor(Math.random() * (to - from + 1) + from);
  return randomTop + '%';
}

// TODO: 待优化
function isLightColor(color) {
  // 解析颜色值为RGB
  const r = parseInt(color.slice(1, 3), 16);
  const g = parseInt(color.slice(3, 5), 16);
  const b = parseInt(color.slice(5, 7), 16);

  // 计算颜色的相对亮度
  const brightness = (r * 0.299 + g * 0.587 + b * 0.114);

  // 判断颜色是否为浅色
  return brightness > 128;
}

function calculateTextColor() {
  return isLightColor ? 'black' : 'white';
}
