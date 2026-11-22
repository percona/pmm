// The collection and indexes every MongoDB case uses.
db = db.getSiblingDB("shop");
db.logs.drop();
db.logs.createIndex({ host: 1, message: "text" });
db.logs.createIndex({ host: 1, severity: 1 });
const docs = [];
for (let i = 0; i < 200; i++) {
  docs.push({
    host: "db" + (i % 10) + ".internal.example",
    severity: i * 50,
    message: i % 7 === 0 ? "checkpoint took long" : "connection accepted",
  });
}
db.logs.insertMany(docs);
