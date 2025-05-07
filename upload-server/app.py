from flask import Flask, request, jsonify
import os

app = Flask(__name__)
UPLOAD_FOLDER = "/cdn-content"
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

@app.route("/upload", methods=["POST"])
def upload_file():
    files = request.files.getlist("files")
    uploaded = []
    for file in files:
        filename = file.filename
        path = os.path.join(app.config['UPLOAD_FOLDER'], filename)
        file.save(path)
        uploaded.append(filename)
    return jsonify({"files": uploaded})

@app.route('/files', methods=['GET'])
def list_files():
    files = os.listdir(UPLOAD_FOLDER)
    return jsonify(files)

app.run(host="0.0.0.0", port=5000)
