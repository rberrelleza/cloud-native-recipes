const express = require("express");
const mongo = require("mongodb").MongoClient;

const app = express();

const url = `mongodb://${process.env.MONGODB_USERNAME}:${encodeURIComponent(process.env.MONGODB_PASSWORD)}@${process.env.MONGODB_HOST}:27017/${process.env.MONGODB_DATABASE}`;

function startWithRetry() {
  mongo.connect(url, { 
    useUnifiedTopology: true,
    useNewUrlParser: true,
    connectTimeoutMS: 1000,
    socketTimeoutMS: 1000,
  }, (err, client) => {
    if (err) {
      console.error(`Error connecting, retrying in 1 sec: ${err}`);
      setTimeout(startWithRetry, 1000);
      return;
    }

    const db = client.db(process.env.MONGODB_DATABASE);

    app.listen(8080, () => {
      app.get("/api/healthz", (req, res, next) => {
        res.sendStatus(200)
        return;
      });

      app.get("/api/recipes", (req, res, next) => {
        console.log(`GET /api/recipes`)
        db.collection('recipes').find().toArray( (err, results) =>{
          if (err){
            console.log(`failed to query recipes: ${err}`)
            res.json([]);
            return;
          }
          res.jsonp(results);
        });
      });

      app.get("/api/recipes/:id", (req, res, next) => {
        const recipeId = req.params.id;
        console.log(`GET /api/recipes/${recipeId}`)
        
        db.collection('recipes').findOne({"id": recipeId}, function(err, doc){
          if (err){
            console.log(`failed to query recipe: ${err}`)
            res.status(500);
            return;
          }

          if (!doc) {
            console.log(`recipe ${recipeId} not found`)
            res.status(404).jsonp({});
            return;
          }

          res.jsonp(doc);
        });
      });

      app.post("/api/recipes/:id/up", (req, res, next) => {
        const filter = { id: req.params.id };
        const options = { upsert: false };
        const updateDoc = {$inc: {upVotes: 1}};
        console.log(`POST /api/recipes/${req.params.id}/up`)

        db.collection('recipes').findOneAndUpdate(filter, updateDoc, options, (err, doc) => {
          if (err){
            console.log(`failed to update recipe: ${err}`)
            res.status(500);
            return;
          }

          if (!doc.value){
            console.log(`POST /api/recipes/${req.params.id}/up not found`)
            res.status(404);
            return;
          }
          res.jsonp(doc.value);
          
        });
      });

      app.post("/api/recipes/:id/down", (req, res, next) => {
        console.log(`POST /api/recipes/${req.params.id}/down`)
        const filter = { id: req.params.id };
        const options = { upsert: false };
        const updateDoc = {$inc: {downVotes: 1}};

        db.collection('recipes').findOneAndUpdate(filter, updateDoc, options, (err, doc) => {
          if (err){
            console.log(`failed to update recipe: ${err}`)
            res.status(500);
            return;
          }

          if (!doc.value){
            console.log(`POST /api/recipes/${req.params.id}/down not found`)
            res.status(404);
            return;
          }
          res.jsonp(doc.value);
          
        });
      });

      

      console.log("Server running on port 8080.");
    });
  });
};

startWithRetry();