document.addEventListener('DOMContentLoaded', function () {
  const chatMessages = document.getElementById('chat-messages');
  const messageInput = document.getElementById('message-input');

  messageInput.focus();
  messageInput.addEventListener('keypress', function (event) {
    if (event.key === 'Enter') {
      event.preventDefault();
      submitInput();
    }
  });

  function appendMessage(sender, message) {
    var messageDiv = document.createElement('div');
    messageDiv.textContent = sender + ': ' + message;
    chatMessages.appendChild(messageDiv);
  }

  async function submitInput() {
    const text = messageInput.value;
    if (text.trim() !== '') {
      // appendMessage('You', text);
      messageInput.value = '';
      const response = await chrome.runtime.sendMessage({ text });
      console.log(response);
    }
  }
});

chrome.commands.onCommand.addListener(function (command) {
  if (command === "_execute_browser_action") {
    chrome.runtime.openPopup();
  }
});
