fmt:
	go fmt ./...
	go vet ./...
	goimports -w ./*.go

test:
	go test -v ./...

integration-test:
	go test -v ./... -tags smoke

dev-up:
	docker run --name mysql_dev -p 33306:3306 -e MYSQL_ROOT_PASSWORD=root -d mysql
	sleep 5
	sudo docker exec -it mysql_dev mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS testdb;"

dev-down:
	docker stop mysql_dev
	docker rm mysql_dev

dev-sql:
	sudo docker exec -it mysql_dev mysql -uroot -proot