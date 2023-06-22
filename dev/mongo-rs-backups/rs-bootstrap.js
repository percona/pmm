var conn, attempt;
while (conn === undefined) {
    try {
        conn = new Mongo("localhost:27017");
    } catch (Error) {
        attempt++;
    }

    if (attempt >= 50) {
        print("Max connection attempts exceeded.");
        break;
    }
    sleep(100);
}

DB = conn.getDB("admin");
DB.runCommand({
    replSetInitiate: {
        _id: "rs0",
        members: [{
                _id: 0,
                host: "mongo1:27017"
            },
            {
                _id: 1,
                host: "mongo2:27017"
            },
            {
                _id: 2,
                host: "mongo3:27017"
            }
        ]
    }
});
