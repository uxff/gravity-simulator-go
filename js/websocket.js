var defaultHost = window.document.location.host || '127.0.0.1:8081';
var wsUri = "ws://"+defaultHost+"/orbs";//host mc.lo redirect to 127.0.0.1
var sendVal = 'cmd=orbs&k=thelist1';
var MyWebsocket = {
    wsUri: wsUri,
    websocket: null,
    wsOk: false,
    lastSendData: null,
    lastRecvData: null,
    lastError: null,
    output: null,
    receiveCallback: null,
    initWebsocket: function () {
        this.output = document.getElementById("ws-msg");
        this.testWebSocket();
    },

    testWebSocket: function () {
        this.websocket = new WebSocket(this.wsUri);
        //console.log(this.websocket);
        this.websocket.onopen = function(evt) {
            MyWebsocket.onOpen(evt);
        };
        this.websocket.onclose = function(evt) {
            MyWebsocket.onClose(evt);
        };
        this.websocket.onmessage = function(evt) {
            MyWebsocket.onMessage(evt);
        };
        this.websocket.onerror = function(evt) {
            MyWebsocket.onError(evt);
        };
    },

    onOpen: function (evt) {
        //console.log(evt);
        this.writeToScreen("CONNECTED");
        this.doSend("WebSocket test if connected");
    },

    onClose: function (evt) {
        this.writeToScreen("DISCONNECTED");
    },

    onMessage: function (evt) {
        try {
            let data = eval('('+evt.data+')');
            this.lastRecvData = evt.data;
            //console.log(data);
            if (this.receiveCallback != undefined) {
                this.receiveCallback(data);
            }
        } catch (e) {
            console.log(e);
        }
        //for (let i in data.data.list) {
        //    //writeToScreen(data.data.list[i].id);
        //}
        //websocket.close();
    },

    onError: function (evt) {
        this.writeToScreen('<span style="color: red;">ERROR:</span> '+ evt.data);
    },

    doSend: function (message) {
        //console.log(websocket.readyState == websocket.CLOSED);
        if (this.websocket && this.websocket.readyState == this.websocket.OPEN) {
            //this.writeToScreen("SENT: " + message);
            this.websocket.send(message);
            this.lastSendData = message;
        } else {
            this.writeToScreen("try to connect websocket failed!");
        }
    },

    writeToScreen: function (message) {
        let pre = document.createElement("p");
        pre.style.wordWrap = "break-word";
        //pre.innerHTML = message;
        //output.appendChild(pre);
        this.output.innerHTML = message || '';
    },

    getOrbList: function () {
    }
};
