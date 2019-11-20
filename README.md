# Documentation for Percona Monitoring and Management (PMM)

This repository is for [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html)

We're looking forward to the new contributors. Instructions will help you. 

## Issues

You can improve any page or section of the documentation.

We use [JIRA](https://jira.percona.com/projects/PMM/issues) to track issues. You can use the Component Documentation filter for issues: [Jira – PMM - Documentation](https://jira.percona.com/issues/?jql=project+%3D+PMM+AND+component+%3D+Documentation).

If you want to add something new or propose changes, please create a task in JIRA.


## Install

Install and build documentation locally as follows:

1.	Installing Sphinx. We use Sphinx-doc v.1+ on production. If you can, install 1.6+, but a higher version will do. [Official instructions](https://www.sphinx-doc.org/en/master/usage/installation.html) 

	For Mac

		brew install sphinx-doc

		export PATH="/usr/local/opt/sphinx-doc/bin:$PATH"

	Check the installation:

		sphinx-build --version
		sphinx-build 2.2.

2.	Make a fork of the pmm-doc repository, then make git clone of your repository locally.

3.	Run the documentation build. The command to prepare html version is the following one:
		
		make html

	**Note:** if you are on BSD-based systems, you may need to comment on the line `@sed -i 's/{{ toc }}/{ toctree\(false\)"` in Makefile.

	Check result:

		copying static files... ... done
		copying extra files... done
		dumping search index in English (code: en)... done
		dumping object inventory... done
		build succeeded, 1319 warnings.

		The HTML pages are in build/html.

		Build finished. The HTML pages are in build/html.

	You can see a lot of Warnings. This is normal.

4.	Now compiled documentation is located in the `/build/html/` folder.
	
	You can simply open "/build/html/index.html" in your browser. 

	Or use Docker

		docker run -dit --name my-apache-app -p 8080:80 -v "$PWD"/build/html/:/usr/local/apache2/htdocs/ httpd:2.4

5.	Check the documentation. It's going to be built without make-up because it's going to use percona.com's make-up.

	![Result](/images/img-readme-result.png)


## WorkFlow

1.	Select or create an issue in [Jira](https://jira.percona.com/issues/?jql=project+%3D+PMM+AND+component+%3D+Documentation)

2.	Make a fork of the pmm-doc repository

3.	Make a separate branch for your issue

4.	Make changes. Use the syntax and examples from existing pages.

5.	Make a local build and make sure the build process was completed successfully.

6.	Check that you made the right commit. Just make a Pull Request to your fork repository.

7.	Make a Pull Request to the pmm-doc repository.

8.	Get recognition and SWAG as a gift.

