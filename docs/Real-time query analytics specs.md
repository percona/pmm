# **Real-Time Query Analytics for MySQL, MongoDB, and PostgreSQL in PMM**

## **1\. Overview**

This document outlines the technical specification for implementing a real-time query analytics feature in Percona Monitoring and Management (PMM). This feature will provide users with immediate insights into query performance for MySQL, MongoDB, and PostgreSQL databases, allowing for faster identification and resolution of performance bottlenecks.

### **1.1. User Perspective**

From a user's perspective, real-time query analytics will appear as a new tab or a "Live View" option within the existing Query Analytics section of the PMM interface. When enabled, this view will display a continuously updating stream of data for both **currently executing** and **recently completed** queries. This provides a complete, near-instantaneous look at what is happening on the database server. Users will be able to see high-frequency queries, long-running queries, and other performance-related metrics as they happen. This will be invaluable for troubleshooting active performance issues.

## **2\. Data Sources**

To provide a comprehensive real-time view, we will pull data from two types of sources for each database: one for currently running queries and another for aggregated statistics of recently completed queries.

* **MySQL:**  
  * **Running Queries:** The performance\_schema.events\_statements\_current table or processlist will be used to identify queries that are currently executing.  
  * **Finished Queries:** The performance\_schema.events\_statements\_summary\_by\_digest table will be the primary data source for aggregated statistics on completed queries. To get the delta for real-time view without disrupting QAN, the PMM Client will take a snapshot of the cumulative statistics at each interval and calculate the difference from the previous snapshot.  
* **MongoDB:**  
  * **Running Queries:** The db.currentOp() command (or the $currentOp aggregation stage) will be used to view in-progress operations.  
  * **Finished Queries (Slow):** The profile command will be used to access the system.profile collection. This will be enabled with a profiling level of 1 to only capture slow queries, minimizing performance impact.  
* **PostgreSQL:**  
  * **Running Queries:** The pg\_stat\_activity view will be the data source for currently active backend processes and their queries.  
  * **Finished Queries:** The pg\_stat\_statements view will continue to be used for tracking execution statistics of all completed SQL statements.

## **3\. Data Collection and Frequency**

To minimize the impact on the monitored database, data will be collected at a configurable interval, with a default of every **1 second**. The collection frequency should be adjustable by the user to balance the need for real-time data with the potential for overhead.

An adaptive polling mechanism will be implemented. If the PMM agent detects a high load on the database, it will automatically reduce the polling frequency to avoid exacerbating the issue.

## **4\. Data Transmission and Flow**

This section describes the data flow, which involves a new **real-time analytics agent** on the PMM Client side and a **real-time analytics server** component on the PMM Server side.

### **4.1. Agent to Server Communication**

The **real-time analytics agent** on the monitored node will collect and send query data to the **real-time analytics server**.

* **Protocol:** This communication will use the existing **protobuf** protocol that PMM uses for other data exchange.  
* **Endpoint:** To avoid overwhelming the existing communication channel used for metrics, a separate, dedicated gRPC endpoint will be established for real-time query data. This will ensure that the flow of regular monitoring data is not interrupted.

### **4.2. Server to UI Communication**

The **real-time analytics server** will act as the API for the frontend, making the data available to the user's browser.

* **Current Implementation (Short Polling):** Since WebSockets are not currently supported, a **short polling** mechanism will be used.  
  1. The UI will make a standard HTTP/S request to a new API endpoint on the PMM Server (e.g., /v1/realtime/query-data).  
  2. The real-time analytics server will immediately respond with the latest batch of in-memory query data it has received.  
  3. Upon receiving the response, the UI will update the view and immediately send another request to the same endpoint.  
* **Future Improvement (WebSockets):** While short polling is a viable initial approach, the long-term goal should be to implement a WebSocket-based solution. WebSockets would provide a more efficient, persistent, and lower-latency connection between the server and the UI, eliminating the overhead of repeated HTTP requests and providing a true real-time experience.  
* **Data Flow:** Regardless of the transport mechanism, the data flow remains the same: The real-time analytics server receives data from the agent, processes it **in-memory**, serves it to the UI, and then discards the data for that interval.

## **5\. Data Persistence**

The data collected for the real-time query analytics feature will **not** be persistently stored in the PMM database (ClickHouse). This feature is designed to be an **ephemeral, in-memory stream** for live monitoring only.

The reasons for this architectural decision are:

* **Performance:** Storing high-frequency (e.g., 1-second interval) data for all queries would generate a massive data volume, placing a significant and unnecessary load on the PMM Server and its database.  
* **Redundancy:** The standard Query Analytics (QAN) feature is already responsible for storing aggregated, historical query data in ClickHouse. The real-time view is a supplement, not a replacement.

## **6\. Data Fields**

The following fields will be collected from the respective data sources.

### **6.1. Common Fields**

| Field Name | Description |
| :---- | :---- |
| query\_id | A unique identifier for the query (e.g., a hash of the query text). |
| query\_text | The full text of the SQL query or command. |
| state | The current state of the query (e.g., running, waiting, finished). |
| execution\_count | The number of times the query has been executed (from finished query source). |
| current\_execution\_time | The elapsed time for a currently running query. |
| total\_execution\_time | The total time spent executing the query (from finished query source). |
| avg\_execution\_time | The average execution time of the query (from finished query source). |
| max\_execution\_time | The maximum execution time of the query (from finished query source). |
| rows\_examined | The number of rows/documents examined by the query. |
| rows\_sent | The number of rows/documents sent to the client. |
| timestamp | The timestamp of the data collection. |

### **6.2. Database-Specific Fields**

#### **MySQL**

| Field Name | Description | Source |
| :---- | :---- | :---- |
| lock\_time | The total time spent waiting for locks. | events\_statements\_current, events\_statements\_summary\_by\_digest |
| tmp\_tables | The number of in-memory temporary tables created. | events\_statements\_summary\_by\_digest |
| tmp\_disk\_tables | The number of on-disk temporary tables created. | events\_statements\_summary\_by\_digest |

#### **MongoDB**

| Field Name | Description | Source |
| :---- | :---- | :---- |
| opid | The operation ID. | db.currentOp() |
| secs\_running | The duration the operation has been running. | db.currentOp() |
| plan\_summary | A summary of the query plan. | system.profile |

#### **PostgreSQL**

| Field Name | Description | Source |
| :---- | :---- | :---- |
| wait\_event\_type | The type of event the backend is waiting for. | pg\_stat\_activity |
| wait\_event | The specific event the backend is waiting for. | pg\_stat\_activity |
| shared\_blks\_hit | The number of shared block cache hits. | pg\_stat\_statements |
| shared\_blks\_read | The number of shared blocks read from disk. | pg\_stat\_statements |

## **7\. Query Display and Configuration**

### **7.1. UI Presentation: Fingerprints vs. Full Query Text**

To provide both a high-level overview and detailed diagnostic capabilities, the UI will adopt a two-tiered approach:

* **Default View (Fingerprints):** The main real-time view will display a list of **query fingerprints**. A fingerprint is a normalized version of a query that abstracts away literal values (e.g., SELECT \* FROM users WHERE id \= ?). This approach is essential for reducing noise and identifying performance patterns, as it groups thousands of similar queries into a single, aggregated entry.  
* **Drill-Down (Full Query Text):** Users will be able to click on any fingerprint in the list. This action will reveal a detailed view showing examples of the **actual, full-text queries** that match that fingerprint. This is critical for debugging, as it allows developers to see the specific parameter values that may be causing performance issues and to copy the exact query for analysis (e.g., running EXPLAIN).

This UI flow will be the standard behavior and not a toggleable option, as it provides the most effective workflow for both monitoring and troubleshooting.

### **7.2. Security Configuration: Disabling Full Query Text Capture**

For environments with strict data privacy or PII (Personally Identifiable Information) requirements, a configuration option will be introduced to prevent the capture and transmission of full query text.

* **Configuration:** A new setting (e.g., disable\_full\_query\_text\_collection) will be available at the PMM agent level.  
* **Behavior:** When this setting is enabled, the PMM client will still collect and send the query fingerprints and all associated metrics. However, the query\_text field will be omitted. In the PMM UI, the fingerprint view will function normally, but clicking on a fingerprint will not show any full-text examples. Instead, a message will indicate that full query text collection is disabled.

## **8\. Security and Access Control**

### **8.1. Label-Based Access Control (LBAC)**

The real-time query analytics feature must adhere to the existing label-based access control (LBAC) model within PMM. The access control will be enforced on the **real-time analytics server** to ensure that users can only view real-time query data for the services they are authorized to see.

* **Enforcement Point:** The filtering logic will reside on the real-time analytics server. The agent will remain unaware of the access control rules and will send all collected data for the monitored node.  
* **Workflow:**  
  1. The UI initiates a short polling request to the real-time API endpoint.  
  2. The real-time analytics server receives the request and identifies the authenticated user making the call.  
  3. Before responding, the server filters the in-memory batch of real-time query data. It compares the labels associated with the data's source (e.g., node name, service name) against the user's permissions.  
  4. Only the data from sources that the user is permitted to view will be included in the HTTP response sent back to the UI.  
  5. Data from restricted sources will be discarded and never sent to the user's browser.

This server-side filtering ensures that the security model is robust and that no unauthorized data is exposed to the client.

## **9\. Performance Considerations**

The real-time query analytics feature is designed to have a minimal impact on the performance of the monitored databases. The following measures will be taken to ensure this:

* **Low-Overhead Data Sources:** The chosen data sources are known to have a low performance overhead.  
* **Adaptive Polling:** The polling frequency will be automatically adjusted based on the database load.  
* **Efficient Data Transmission:** Data will be compressed to reduce network bandwidth.  
* **Server-Side Aggregation:** The PMM Server will handle the aggregation and processing of the real-time data, offloading this work from the monitored database.

## **10\. Enable/Disable Mechanism**

The real-time query analytics feature will be disabled by default and controlled centrally by pmm-managed.

* **Control Flow:**  
  1. A user enables or disables the feature for a specific service via a toggle switch in the PMM UI.  
  2. This action instructs pmm-managed on the PMM Server to update the configuration for that service.  
  3. pmm-managed sends the updated configuration to the pmm-agent on the corresponding monitored node.  
  4. Based on this information, pmm-agent will either start or stop the appropriate **real-time analytics agent** process.

When the feature is disabled, the real-time analytics agent will not run, ensuring that no data is collected or transmitted, and thus there is no performance impact on the database.

## **11\. Open Questions**

The following questions should be addressed before or during the implementation phase:

1. **UI/UX Visualization:** How should the real-time data be visualized beyond a simple table? Should there be sparklines, top-N graphs, or other visual elements to quickly identify trends? What sorting and filtering options should be available to the user?  
2. **Scalability and Performance Thresholds:** What are the specific thresholds for the "adaptive polling" mechanism? How do we define "high load" for each database type? What is the expected memory and CPU footprint of the real-time analytics server per monitored service, and what are the scalability limits?  
3. **MongoDB Finished Query Data:** The current approach for finished queries in MongoDB only captures slow queries (system.profile level 1). This means we will miss high-frequency, fast queries that could still contribute significantly to load. Is this acceptable, or do we need an alternative strategy for a more complete view?  
4. **Error Handling and Reporting:** How will the agent and server handle scenarios where a required data source (e.g., pg\_stat\_statements) is not enabled on the target database? How will this state be communicated to the user in the UI?  
5. **Short Polling vs. Long Polling:** Is short polling the most efficient interim solution? Should we consider long polling to reduce the number of immediate, repeated HTTP requests and lower the overhead on the server before WebSockets are implemented?  
6. **Scope of Data Capture:** Should the initial version capture both running and finished queries as specified, or should we focus on only one (e.g., just currently running queries) for a simpler MVP?  
7. **Short-Term History:** While long-term persistence is out of scope, should the real-time analytics server keep a small, in-memory history (e.g., the last 1-5 minutes of data) to provide slightly more context when a user opens the page?  
8. **Inventory UI Integration:** How should the status of the real-time analytics agent (enabled/disabled) be presented on the main Inventory page for a given service?  
9. **Implementation Rollout:** Will this feature be developed and released for all three databases (MySQL, MongoDB, PostgreSQL) simultaneously, or will it be a phased rollout (e.g., MongoDB first)? If phased, what is the timeline for the other databases?
