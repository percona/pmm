.. _pmm.qan.mongodb:

--------------------------------------------------------------------------------
`MongoDB specific <pmm.qan.mongodb>`_
--------------------------------------------------------------------------------

Query Analytics for MongoDB
================================================================================

|mongodb| is conceptually different from relational database management systems,
such as |mysql| or |mariadb|. Relational database management systems store data
in tables that represent single entities. In order to represent complex objects
you may need to link records from multiple tables. |mongodb|, on the other hand,
uses the concept of a document where all essential information pertaining to a
complex object is stored together.

.. _figure.pmm.qan.mongodb.query-summary-table.mongodb:

.. figure:: .res/graphics/png/qan.query-summary-table.mongodb.1.png

   A list of queries from a |mongodb| host

|qan| supports monitoring |mongodb| queries. Although |mongodb| is not a relational
database management system, you analyze its databases and collections in the
same interface using the same tools. By using the familiar and intuitive
interface of :ref:`QAN <QAN>` you can analyze the efficiency of your application
reading and writing data in the collections of your |mongodb| databases.

.. seealso:: 

   What |mongodb| versions are supported by |qan|?
      :ref:`See more information about how to configure MongoDB <pmm.conf.mongodb.supported-version>`


.. _figure.pmm.qan.mongodb.query-metrics:

.. figure:: .res/graphics/png/qan.query-metrics.mongodb.1.png

   Analyze |mongodb| queries using the same tools as relational database
   management systems.

.. include:: .res/replace.txt
