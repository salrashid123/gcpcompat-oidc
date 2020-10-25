var firebase = require("firebase/app");
require("firebase/auth");

const uid = "alice@domain.com";
const password = 'Password1'

var firebaseConfig = {
  apiKey: "AIzaSyAf7wesN7auBeyfJQJs5d_QfT24kMH7OG8",
  authDomain: "cicp-oidc-test.firebaseapp.com",
  projectId: "cicp-oidc-test",
  appId: "cicp-oidc-test",
};

firebase.initializeApp(firebaseConfig);

firebase.auth().signInWithEmailAndPassword(uid, password).then(result => {
  console.log(JSON.stringify(result, null, 2))
}).catch(function(error) {
    // Handle Errors here.
    var errorCode = error.code;
    var errorMessage = error.message;
    console.log(errorMessage);
  });