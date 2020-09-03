# MongoDB RocksDB Details

![image](../_images/PMM_MongoDB_RocksDB_Details.jpg)

## Document Activity

Mixed metrics: Docs per second inserted, updated, deleted or returned on any type of node (primary or secondary); + replicated write Ops/sec; + TTL deletes per second.

## Client Operations

Ops and Replicated Ops/sec, classified by legacy wire protocol type (query, insert, update, delete, getmore).

## Queued Operations

Operations queued due to a lock.

## Scanned and Moved Objects

This panel shows the number of objects (both data (scanned_objects) and index (scanned)) as well as the number of documents that were moved to a new location due to the size of the document growing. Moved documents only apply to the MMAPv1 storage engine.

## Page Faults

Unix or Window memory page faults. Not necessarily from mongodb.

