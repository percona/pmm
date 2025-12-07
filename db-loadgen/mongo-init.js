// MongoDB initialization script
// This script creates a test database and user for the load generator

// Switch to loadtest database (match what the Go code expects)
db = db.getSiblingDB('loadtest');

// Create a test user with read/write permissions for the load generator
db.createUser({
  user: "testuser",
  pwd: "testpass",
  roles: [
    {
      role: "readWrite",
      db: "loadtest"
    }
  ]
});

// Switch to admin database to create PMM monitoring user
db = db.getSiblingDB('admin');

// Create a PMM user with necessary monitoring privileges
db.createUser({
  user: "pmm-mongodb",
  pwd: "pmm-pass",
  roles: [
    { role: "clusterMonitor", db: "admin" },
    { role: "read", db: "admin" },
    { role: "read", db: "local" },
    { role: "read", db: "config" },
    { role: "read", db: "loadtest" }
  ]
});

// Switch back to loadtest database
db = db.getSiblingDB('loadtest');

// Create a test collection to ensure database exists
db.load_test.insertOne({
  name: "init",
  value: 0,
  created_at: new Date(),
  _temp: true
});

// Remove the temporary document
db.load_test.deleteOne({ _temp: true });

print("MongoDB loadtest database and PMM user initialized successfully"); 