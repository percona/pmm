.. _pmm/release/1-2-1:

|pmm.name| |release| 
********************************************************************************

:Date: August 16, 2017

For install and upgrade instructions, see :ref:`deploy-pmm`.

This hotfix release improves memory consumption


Changes in |pmm-server|
================================================================================

The following changes were introduced in *PMM Server* 1.2.1:

.. rubric:: Bug fixes

* :pmmbug:`1280`: PMM server affected by nGinx
  `CVE-2017-7529 <https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-7529>`_

  An integer overflow exploit could result in a DOS (Denial of Service) for the
  affected nginx service with the ``max_ranges`` directive not set.

  This problem is solved by setting the ``set max_ranges`` directive to 1 in the
  nGinx configuration.

.. rubric:: Improvements


* :pmmbug:`1232`: Update the default value of the ``METRICS_MEMORY``
  configuration setting

  Previous versions of *PMM Server* used a different value for the METRICS_MEMORY
  configuration setting which allowed Prometheus to use up to 768MB of memory.

  |pmm-server| 1.2.0 used the storage.local.target-heap-size setting, its default
  value being 256MB. Unintentionally, this value reduced the amount of memory that
  Prometheus could use.  As a result, the performance of Prometheus was
  affected.

  To improve the performance of Prometheus, the default setting of
  ``storage.local.target-heap-size`` has been set to 768 MB.

.. |release| replace:: 1.2.1

.. include:: .res/replace/name.txt
