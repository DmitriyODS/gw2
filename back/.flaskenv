FLASK_APP=app:create_app
UPLOAD_FOLDER=./uploads
DATABASE_URL=postgresql://grovework:grovework_local@localhost:5432/grovework
REDIS_URL=redis://localhost:6379/0
JWT_SECRET_KEY=dev-jwt-secret-key-min-32-chars-local-xxxx
SECRET_KEY=dev-flask-secret-key-min-32-chars-local-xxxx
AI_KEY_ENCRYPTION_KEY=X3hFOVZ6XbAzlaygv2PfLbnmBIaH373CK8MqrrAhr8k=
YOUGILE_ENC_KEY=CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=
# gRPC-адрес Go-микросервиса звонков (callsvc на хосте, см. dev.sh /
# make dev-calls). LiveKit-ключи Flask больше не нужны — ими владеет callsvc.
CALLS_GRPC_ADDR=localhost:9090
