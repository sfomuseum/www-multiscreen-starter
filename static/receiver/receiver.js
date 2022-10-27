window.addEventListener("load", function load(event){

    var messages_el = document.getElementById("messages");
    
    var root_url = location.protocol + "//" + location.host;

    var sender_url = root_url + "/";
    var sse_url = root_url + "/sse/";
    var code_url = root_url + "/code/";
    
    // initialize the map
    
    // initialize SSE stuff

    var ev = new EventSource(sse_url);

    ev.onopen = function(e){
	console.log("SSE connected");
    };

    ev.onerror = function(e){
	console.log("SSE error", e);
    };
    
    ev.onmessage = function(e) {

	try {
	    var msg = JSON.parse(e.data);
	} catch (err) {
	    console.log("Failed to parse message", e.data, err);
	    return;
	}

	if (msg.type == "update"){

	    var dt = new Date();
	    
	    var item = document.createElement("li");
	    item.innerText = dt.toLocaleString() + ": " + msg.data.body;

	    messages_el.prepend(item);
	    
	} else if (msg.type == "showCode"){
	    
	    var code = msg.data;

	    var url = sender_url + "?code=" + encodeURIComponent(msg.data.code);
	    console.log("URL", url);
	    
	    var qr_el = document.getElementById("qr");
	    qr_el.innerHTML = "";

	    var qr_args = {
		height: 150,
		width: 150,
		text: url,
	    }
	    
	    new QRCode(qr_el, qr_args);

	    qr_el.style.display = "block";

	    var url_el = document.getElementById("url");
	    url_el.setAttribute("href", url);
	    url_el.innerText = url;
	    
	} else if (msg.type == "hideCode"){

	    var qr_el = document.getElementById("qr");	    
	    qr_el.innerHTML = "";

	    qr.style.display = "none";
	    
	    var url_el = document.getElementById("url");	    
	    url_el.innerHTML = "";
	    url_el.setAttribute("href", "#");
	    
	} else {
	    console.log("Unhandled message type", msg.type)
	}
	
    }

    // Fetch the most recent access code to display

    setTimeout(function(){

	var on_load = function(rsp){
	    console.log("WHAT", rsp);
	};
	
	var req = new XMLHttpRequest();
	
	req.addEventListener("load", on_load);
	req.open("GET", code_url, true);
	req.send();
	
    }, 500);
});
