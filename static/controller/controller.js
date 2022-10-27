/* */

window.addEventListener("load", function load(event){

    var message_el = document.getElementById("message");
    var feedback_el = document.getElementById("feedback");    
    var send_btn = document.getElementById("send");

    var feedback = function(msg){
	feedback_el.innerHTML = "";
	feedback_el.innerText = msg;
    };
    
    var ws_url = "ws://" + location.host + "/ws/";
    
    var params = new URLSearchParams(window.location.search);
    var code = params.get("code");
    
    // initialize WS stuff

    var socker = null;
    var connected = false;

    if (!code){
	feedback("Missing code");
	send_btn.setAttribute("disabled", "disabled");
	return;
    }
	
    socket = new WebSocket(ws_url);
    
    socket.onopen = function(e){
	console.log("connected", e);
	connected = true;
    };
    
    socket.onclose = function(e){
	console.log("close");
	connected = false;
    }
    
    socket.onerror = function(e){
	feedback("Socket closed");
	send_btn.setAttribute("disabled", "disabled");	
	console.log("error", e);
	// connected = false;
    }
    
    socket.onmessage = function(rsp){
	
	var data = rsp['data'];
	console.log("received", data);

	if (data == "invalid"){
	    feedback("Invalid");
	} else if (data == "expired"){
	    feedback("Code has expired");
	} else if (data == "relay"){
	    feedback("Message relayed '" + message_el.value + "'");
	    message_el.value = "";
	}
	
    };

    send_btn.onclick = function(){

	var msg = message_el.value;

	if (! msg){
	    feedback("Nothing to send.");
	    return false;
	}

	feedback("");
	
	var update_msg = {
	    "type": "update",
	    "code": code,
	    "body": msg,
	};
	
	socket.send(JSON.stringify(update_msg));
	return false;
    };
    
});
