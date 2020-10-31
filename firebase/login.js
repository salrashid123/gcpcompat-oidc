var firebase = require("firebase/app");
require("firebase/auth");

const uid = "alice@domain.com";
const password = 'Password1'

var firebaseConfig = {
  apiKey: "AIzaSyDxRf3ggNGHO8z6HicMCGq8xY",
  authDomain: "mineral-minutia-820.firebaseapp.com",
  projectId: "mineral-minutia-820",
  appId: "mineral-minutia-820",
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