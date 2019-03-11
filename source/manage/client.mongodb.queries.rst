.. _pmm-admin.add-mongodb-queries:

`Understanding MongoDB query analytics service <pmm-admin.add-mongodb-queries>`_
================================================================================

Use the |opt.mongodb-queries| alias to enable |mongodb| query analytics.

.. _pmm-admin.add-mongodb-queries.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add-mongodb-queries:

.. include:: ../.res/code/pmm-admin.add.mongodb-queries.txt
		 
This creates the ``pmm-mongodb-queries-0`` service
that is able to collect |qan| data for multiple remote |mongodb| server instances.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.add-mongodb-queries.options:

.. rubric:: OPTIONS

The following options can be used with the |opt.mongodb-queries| alias:

|opt.uri|
  Specify the |mongodb| instance URI with the following format::

   [mongodb://][user:pass@]host[:port][/database][?options]

  By default, it is ``localhost:27017``. 

  .. important::

     In cases when the password contains special symbols like the *at* (@)
     symbol, the host might not not be detected correctly. Make sure that you
     insert the password with special characters replaced with their escape
     sequences. The simplest way is to use the :code:`encodeURIComponent` JavaScript function.
     
     For this, open the web console of your browser (usually found under
     *Development tools*) and evaluate the following expression, passing the
     password that you intend to use:

     .. code-block:: javascript

	> encodeURIComponent('$ecRet_pas$w@rd')
	"%24ecRet_pas%24w%40rd"

     .. admonition:: |related-information|

	MDN Web Docs: encodeURIComponent
	   https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

.. include:: ../.res/contents/note.option.mongodb-queries.txt

For more information, run
|pmm-admin.add|
|opt.mongodb-queries|
|opt.help|.

.. seealso::

   Default ports
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`


.. include:: ../.res/replace.txt
