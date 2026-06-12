FLASK_APP=app:create_app
UPLOAD_FOLDER=./uploads
DATABASE_URL=postgresql://grovework:grovework_local@localhost:5432/grovework
REDIS_URL=redis://localhost:6379/0
# Публичный ключ PASETO (пара к dev-ключу authsvc в dev.sh / make dev-auth).
# Токены выпускает authsvc, Flask только проверяет подпись.
PASETO_PUBLIC_KEY=15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1
SECRET_KEY=dev-flask-secret-key-min-32-chars-local-xxxx
# gRPC-адреса Go-микросервисов (на хосте, см. dev.sh / make dev-calls /
# make dev-messenger): Flask-шлюз зовёт только callsvc (ринг-фаза) и
# msgsvc (плашки звонков).
CALLS_GRPC_ADDR=localhost:9090
MESSENGER_GRPC_ADDR=localhost:9092
