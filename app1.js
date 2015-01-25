// The workhorse JS file.
//
// The manifest is checked to see if the app is installed.
var MANIFEST = "http://"+HOST+"/manifest.webapp";


// PUT the registration back to the server.
function putRegis(endpoint) {
    console.log("putting ", endpoint, HOST);
    try {
        // Currently only FormData is correctly posted back to server.
        var form = FormData();
        form.append("endpoint", endpoint);
        // "mozSystem" needs to be included and defined in the permissions
        // in the manifest.webapp file.
        var post = new XMLHttpRequest({mozSystem:true});
        post.open("POST", "http://"+HOST+"/register.php", true);
        post.send(form)
    } catch (e) {
        console.error("putRegis:", e);
    }
}

// check to see if the user has registered for an update.
function sendRegister() {
    // if there's a cookie set, don't bother re-registering.
    if (! docCookies.hasItem("pr")) {
        // Check the currently list of push registrations.
        var regs = navigator.push.registrations();
        regs.onsuccess = function(eregs) {
            console.debug("registrations success", eregs,
                        eregs.constructor.name);
            var req;
            // nothing is registered, so get a new registration.
            if (regs.result.length == 0) {
                req = navigator.push.register();
                req.onsuccess = function(ereq) {
                    // send the endpoint to the app server
                    // this is called in sequence by the app server on
                    // change.
                    console.debug("success",ereq, req);
                    var endpoint = req.result;
                    putRegis(endpoint);
                    console.debug("Check register");
                }
                req.onerror = function(e) {
                    alert(e.constructor.name);
                    console.error("req error ", e.constructor.name, e);
                }
                console.debug("attempting registration", req, req.constructor);
           } else {
               // the app is already registered.
               // Optionally re-register this app (obviously the server should
               // do dupe detection.
                for (i=0;i<regs.result.length;i++) {
                    var endpoint = regs.result[i].pushEndpoint;
                    console.debug("already registered",
                            regs.result[i].pushEndpoint);
                    putRegis(endpoint);
                }
           }
        }
        //set the cookie so we don't re-register.
        docCookies.setItem("pr", 1, Infinity);
        // and hide the bottom panel.
        document.getElementById("webapp").style.display="none";
    }
}

// Something's happened on the server! Display the notification.
function showApp(e) {
    console.debug("notification!", JSON.stringify(e));
    var note = navigator.mozNotification.createNotification("Are We Fired Yet",
            "Someone's getting fired...",
            "http://app.evilonastick.com/img/hr_icon_32.jpg");
    // try to display the app on click.
    note.onclick = function() {
        // TODO: Not sure why this isn't refeshing on launch. May need to
        // add some code to force it.
        navigator.mozApps.getSelf().launch();
    };
    note.show();
}

// Can the client do SimplePush?
function canPush() {
    if (navigator.push) {
        // install the "push" message handler.
        console.debug("Adding handler...");
        if (navigator.mozSetMessageHandler) {
            // Not sure why I have to wrap this call. May have to do with
            // showApp's context not being created when this is assigned.
            navigator.mozSetMessageHandler('push', function(e){showApp(e);});
        }
        if (!docCookies.hasItem("pr")) {
            var pushctl=document.getElementById("pushctl");
            pushctl.style.display="block";
            pushctl.addEventListener("click", sendRegister);
            return;
        }
    }
    document.getElementById("webapp").style.display="none";
}

// Try to install the hosted page as a webapp.
function installApp() {
    // install the app.
    console.debug("Installing....");
    var req = navigator.mozApps.install(MANIFEST);
    req.onerror = function(e) {
        // Badness. Sad panda is sad.
        console.error(e.explicitOriginalTarget.error);
        document.getElementById("webapp").style.display="none";
    };
    req.onsuccess = function(e){
        // Yay! App is installed. check to see if we can push.
        console.debug("install success", e);
        document.getElementById("install").style.display="none";
        canPush();
    };
}

// Are we installed yet?
function isInstalled(e) {
    // Push Notifications are only available to webapps.
    console.debug("isInstalled", e);
    if (null != e.target.result) {
        canPush();
    } else {
        // display "Do you want to install" thingy here.
        document.getElementById("webapp").style.display="block";
        var inst = document.getElementById("install");
        inst.style.display="block";
        inst.addEventListener("click", installApp);
    }
}

// Startup routines.
function startApp() {
    var ctl = document.getElementById("webapp");
    /*
    if (! docCookies.hasItem("pr")) {
        ctl.addEventListener("click",addUpdate);
        ctl.style.display="block";
    }
    */
    if (navigator.mozApps) {
        ctl.style.display="block";
        navigator.mozApps.getSelf().onsuccess = isInstalled;
        navigator.mozApps.getSelf().onerror = function(e) {
              console.debug("getSelf fail", e);
        };
        if (navigator.mozSetMessageHandler) {
        }
        console.debug("mozApps present");
        console.debug(navigator.mozApps.getSelf());
    } else {
        console.error("No mozApps, because you hate the free web.");
    }
}


// General handlers for this app.
window.addEventListener("load", startApp);
