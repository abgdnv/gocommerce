FROM migrate/migrate:4

COPY docker/migrator/run-migrations.sh /usr/local/bin/run-migrations.sh

RUN chmod +x /usr/local/bin/run-migrations.sh

ENTRYPOINT ["/usr/local/bin/run-migrations.sh"]