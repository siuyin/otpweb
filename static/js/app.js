document.addEventListener("DOMContentLoaded", function(event) {
  console.log("DOMContentLoaded");
  
  // get elements
  var email = document.getElementById("email");
  var otp = document.getElementById("otp");
  var siBtn = document.getElementById("si-btn");
  var msg = document.getElementById("msg");

  // sign-in button
  siBtn.addEventListener("click", function(e) {
    console.log("email: "+email.value+" otp: "+otp.value);
    
    var req = new XMLHttpRequest();
    req.open("GET","/otpvldt?email="+email.value+"&otp="+otp.value);
    req.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    req.onreadystatechange = function () {
      if (req.readyState === 4 && req.status === 200) {
        msg.textContent = req.responseText;
        if (req.responseText.indexOf("error:") !== -1) {
          msg.classList.add("error"); 
        } else {
          msg.classList.remove("error");
        }
      }
    };
    req.send();
  });
});
