.. _pqa:

=======================
Percona Query Analytics
=======================

Percona Query Analytics (PQA) enables database administrators and application developers to analyze MySQL queries over periods of time and find performance problems. PQA helps you optimize database performance by making sure that queries are executed as expected and within the shortest time possible. In case of problems, you can see which queries may be the cause and get detailed metrics for them.

  * :ref:`PQA Agent <agent>`: Collects query data and sends it to *PQA Datastore*.

  * :ref:`PQA Datastore <datastore>`: Repository and API for storing and accessing query data collected by *PQA Agent*.

  * :ref:`PQA App <webapp>`: Web application for visualizing query data.

.. toctree::
   :hidden:

   Agent <agent>
   Datastore <datastore>
   Web App <webapp>
