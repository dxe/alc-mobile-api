dev:
	docker-compose up --build

rm_db:
	find . -path "./.dev_db/*" -not -name ".gitignore" -delete

.PHONY: dev rm_db