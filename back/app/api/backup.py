from flask import Blueprint, request, jsonify, send_file, current_app
from app.services import backup_service
from app.utils.permissions import require_role, ADMIN

bp = Blueprint("backup", __name__, url_prefix="/api/backup")


@bp.get("/export")
@require_role(ADMIN)
def export_backup():
    """
    Скачать резервную копию (ZIP-архив).
    ---
    tags: [backup]
    security: [BearerAuth: []]
    responses:
      200:
        description: ZIP-архив с данными и аватарками
        content:
          application/zip: {}
    """
    upload_folder = current_app.config["UPLOAD_FOLDER"]
    buf = backup_service.export_zip(upload_folder)
    return send_file(
        buf,
        mimetype="application/zip",
        as_attachment=True,
        download_name="grovework_backup.zip",
    )


@bp.post("/import")
@require_role(ADMIN)
def import_backup():
    """
    Восстановить из резервной копии (ZIP). ДЕСТРУКТИВНАЯ операция — полная замена данных!
    ---
    tags: [backup]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        multipart/form-data:
          schema:
            type: object
            properties:
              file:
                type: string
                format: binary
    responses:
      200:
        description: Восстановление выполнено
      400:
        description: Файл не передан или повреждён
    """
    if "file" not in request.files:
        return jsonify({"error": "NO_FILE", "message": "Файл не передан"}), 400

    file = request.files["file"]
    zip_bytes = file.read()

    upload_folder = current_app.config["UPLOAD_FOLDER"]
    try:
        backup_service.import_zip(zip_bytes, upload_folder)
    except Exception as e:
        return jsonify({"error": "IMPORT_ERROR", "message": f"Ошибка импорта: {str(e)}"}), 400

    return jsonify({"message": "Данные восстановлены"}), 200
