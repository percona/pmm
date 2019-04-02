FROM ddidier/sphinx-doc

RUN apt-get update
RUN apt-get install wget
RUN python -m pip install sphinx==1.6.7


ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
