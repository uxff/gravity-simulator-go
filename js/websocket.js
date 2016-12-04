var wsUri = "ws://yii2a.lo:9501/up";
var MyWebsocket = {
    wsUri: wsUri,
    websocket: null,
    wsOk: false,
    lastSendData: null,
    lastRecvData: null,
    lastError: null,
    output: null,
    sceneMgr: null,
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
            var data = eval('('+evt.data+')');
            this.lastRecvData = evt.data;
            //console.log(data);
            if (this.sceneMgr.isInited==false) {
                this.sceneMgr.initOrbs(data.data.list);
            } else {
                this.sceneMgr.updateOrbs(data.data.list);
            }
        } catch (e) {
            console.log(e);
        }
        //for (var i in data.data.list) {
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
            this.writeToScreen("SENT: " + message);
            this.websocket.send(message);
            this.lastSendData = message;
        } else {
            this.writeToScreen("websocket not ready!");
        }
    },

    writeToScreen: function (message) {
        var pre = document.createElement("p");
        pre.style.wordWrap = "break-word";
        //pre.innerHTML = message;
        //output.appendChild(pre);
        this.output.innerHTML = message;
    },

    //window.addEventListener("load", init, false);
    //draw(ctx, list);
    //function draw (ctx, list) {
    //    clear(ctx);
    //    //console.log(list);
    //    for (var i=0; i<list.length; ++i) {
    //        if (list[i].lifeStep==1) {
    //            var orb = list[i];
    //            //console.log(orb)
    //            //console.log(orb.size);
    //            ctx.strokeStyle = "hsla(" + 50 + ", 90%, 50%, 1)";
    //            //ctx.shadowColor = "hsla(" + 50 + ", 100%, 55%, 1)";
    //            //ctx.shadowBlur = orb.size * 1;
    //            ctx.beginPath();
    //
    //            ctx.arc(list[i].x, list[i].y, list[i].size, 0, Math.PI*2 , false);
    //
    //            ctx.closePath();
    //            ctx.stroke();
    //        }
    //    }
    //}
    //var key='mcasync2';
    //setInterval('doSend("'+key+'")', 500);
    getOrbList: function () {
        var key='mcasync2';
        //this.
    }
};
