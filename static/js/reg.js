document.addEventListener("DOMContentLoaded", function(event) {
  
  // get elements
  var email = document.getElementById("email");
  var pass1 = document.getElementById("pass1");
  var pass2 = document.getElementById("pass2");
  var regBtn = document.getElementById("reg-btn");
  var chkExists = document.getElementById("existing-chk");
  var msg = document.getElementById("msg");

  var otpFrm = document.getElementById("otp-form");
  var em1 = document.getElementById("email-1");
  var pw1 = document.getElementById("pass-1");
  //
  // hide otp form
  otpFrm.classList.add("hidden");

  // sign-in button
  regBtn.addEventListener("click", function(e) {
    if (pass1.value !== pass2.value) {
      alert("passwords to not match. Pease re-enter");
      return;
    }
    var req = new XMLHttpRequest();
    em1.value = email.value;
    pw1.value = pass1.value;
    console.log("email: "+em1.value+" pw: "+pass1.value+" chk-exists: "+chkExists.checked);
    req.open("POST","/register");
    req.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    req.onreadystatechange = function () {
      if (req.readyState === 4 && req.status === 200) {
        msg.textContent = req.responseText;
        if (req.responseText.indexOf("error:") !== -1) {
          msg.classList.add("error"); 
        } else {
          msg.classList.remove("error");
          otpFrm.classList.remove("hidden");
        }
      }
    };
    req.send("email="+email.value+"&pw="+pass1.value+"&chk-exists="+chkExists.checked);
  });

});
