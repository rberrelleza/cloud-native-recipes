const mongo = require("mongodb").MongoClient;

const url = `mongodb://${process.env.MONGODB_USERNAME}:${encodeURIComponent(process.env.MONGODB_PASSWORD)}@${process.env.MONGODB_HOST}:27017/${process.env.MONGODB_DATABASE}`;

var insert = function(collection, data, resolve, reject) {
  const d = require(data);
  d.forEach((doc) => {
    doc._id = doc.id;
  });
  collection.insertMany(d,{ordered: false},  (err, r) => {
    if (err) {
      if (err.code != 11000) {
        return reject(err);
      }
    } 

    resolve();
  });
}

function load() {
  mongo.connect(url, { 
    useUnifiedTopology: true,
    useNewUrlParser: true,
    connectTimeoutMS: 300,
    socketTimeoutMS: 300,
  }, (err, client) => {
    if (err) {
      console.error(`Error connecting, retrying in 300 msec: ${err}`);
      setTimeout(loadWithRetry, 300);
      return;
    }

    var promises = [];
    const db = client.db(process.env.MONGODB_DATABASE);

    

    promises.push(new Promise((resolve, reject)=>{
      insert(db.collection('recipes'), "./data/recipes.json", resolve, reject);
    }));

    Promise.all(promises)
    .then(function() { 
      console.log('all loaded'); 
      process.exit(0);
    })
    .catch((err) => {
      console.error(`errors while loading: ${err}`);
      process.exit(1);
    });      
  });
};

load();