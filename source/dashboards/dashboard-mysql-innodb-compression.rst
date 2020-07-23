.. _dashboard-mysql-innodb-compression:

########################
MySQL InnoDB Compression
########################

This dashboard helps you analyze the efficiency of InnoDB compression.

.. seealso::

   MySQL Documentation
      https://dev.mysql.com/doc/refman/5.7/en/innodb-information-schema-compression-tables.html


********************************************
Compression level and failure rate threshold
********************************************

.. rubric:: InnoDB Compression Level

The level of zlib compression to use for InnoDB compressed tables and indexes.

.. rubric:: InnoDB Compression Failure Threshold

The compression failure rate threshold for a table.

.. rubric:: Compression Failure Rate Threshold

The maximum percentage that can be reserved as free space within each compressed
page, allowing room to reorganize the data and modification log within the page
when a compressed table or index is updated and the data might be recompressed.

.. rubric:: Write Pages to the Redo Log

Specifies whether images of re-compressed pages are written to the redo
log. Re-compression may occur when changes are made to compressed data.

************************************
Statistics of Compression Operations
************************************

This section contains the following metrics:

- Compress Attempts
- Uncompressed Attempts
- Compression Success Ratio

.. rubric:: Compress Attempts

Number of compression operations attempted. Pages are compressed whenever an
empty page is created or the space for the uncompressed modification log runs
out.

.. rubric:: Uncompressed Attempts

Number of uncompression operations performed. Compressed InnoDB pages are
uncompressed whenever compression fails, or the first time a compressed page is
accessed in the buffer pool and the uncompressed page does not exist.

**************
CPU Core Usage
**************

- CPU Core Usage for Compression
- CPU Core Usage for Uncompression

*****************
Buffer Pool Total
*****************

- Total Used Pages
- Total Free Pages

***************
Buffer Pool All
***************

- Used Pages (Buffer Pool 0)
- Pages Free (Buffer Pool 0)
