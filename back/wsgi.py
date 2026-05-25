import eventlet
eventlet.monkey_patch()

import os
from dotenv import load_dotenv
load_dotenv(".flaskenv")
load_dotenv()
from app import create_app
from app.extensions import socketio

app = create_app()

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5001))
    # debug=False обязательно: при debug=True flask_socketio переключается на
    # werkzeug-сервер, который НЕ поддерживает WebSocket. Auto-reload в dev
    # достигается перезапуском процесса (см. dev-команды в Makefile).
    socketio.run(app, host="0.0.0.0", port=port, debug=False, allow_unsafe_werkzeug=False)
