var admin = require('firebase-admin');
//export GOOGLE_APPLICATION_CREDENTIALS=/path/to/svc_account.json

admin.initializeApp({
    credential: admin.credential.applicationDefault(),
});
  
const uid = "alice@domain.com";
const password = 'Password1'


admin.auth().createUser({
    email: uid,
    uid: uid,
    emailVerified: true,
    password: password,
    displayName: 'alice',
    disabled: false,
  })
    .then(function(userRecord) { 
     // console.log('Successfully created new user:', userRecord.toJSON());
      admin.auth().setCustomUserClaims(uid, {isadmin: 'true', mygroups:["group1","group2"]}).then(() => {
        admin.auth().getUser(uid).then((userRecord) => {
            console.log(userRecord.toJSON());
            process.exit(0);
          });
      });
    })
    .catch(function(error) {
      console.log('Error creating new user:', error);
    });

