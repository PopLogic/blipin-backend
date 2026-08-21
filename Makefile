postgres:
	docker run --name blipin-postgres --network blipin-network -e POSTGRES_USER=root -e POSTGRES_PASSWORD=root -p 5432:5432 -d postgres:17-alpine

createdb: 
	docker exec -it blipin-postgres createdb --username=root --owner=root blipin_client

dropdb:
	docker exec -it blipin-postgres dropdb blipin_client

migrateup:
	migrate -path db/migration -database "postgres://root:root@localhost:5432/blipin_client?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgres://root:root@localhost:5432/blipin_client?sslmode=disable" -verbose down

sqlc:
	sqlc generate

.PHONY: postgres createdb dropdb migrateup migratedown sqlc