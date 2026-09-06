import sys
import traceback
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

import sentencepiece as spm
from ctranslate2 import Translator

# The Go supervisor captures stdout/stderr through a pipe, which on Windows
# defaults to cp1252 and raises UnicodeEncodeError the first time a non-Latin
# translation is printed, killing the request. Force UTF-8.
for stream in (sys.stdout, sys.stderr):
    try:
        stream.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass

PORT = 8000

# --- decoding options -------------------------------------------------------
# beam_size: 1 = greedy (fastest), 4-5 = better wording at a few x the cost.
# max_decoding_length: cap on output tokens; multi-row replicas are joined
#   before translation so keep this well above a single line.
BEAM_SIZE = 4
MAX_DECODING_LENGTH = 512
COMPUTE_TYPE = "int8"  # int8 is the right trade-off on CPU; use "float32" on GPU

langs = {}


def get_model(from_lang, to_lang):
    key = f"{from_lang}_{to_lang}"
    if key not in langs:
        path = f"./{key}/1"
        langs[key] = {
            "translator": Translator(path, compute_type=COMPUTE_TYPE),
            "src": spm.SentencePieceProcessor(f"{path}/{from_lang}.spm.model"),
            "tgt": spm.SentencePieceProcessor(f"{path}/{to_lang}.spm.model"),
        }
        print(f"loaded {from_lang} -> {to_lang} model", flush=True)
    return langs[key]


def translate(from_lang, to_lang, text):
    lines = text.splitlines() or [""]
    model = get_model(from_lang, to_lang)
    tokens = model["src"].EncodeAsPieces(lines)
    result = model["translator"].translate_batch(
        tokens,
        batch_type="tokens",
        beam_size=BEAM_SIZE,
        max_input_length=0,
        max_decoding_length=MAX_DECODING_LENGTH,
    )
    hyps = [item.hypotheses[0] for item in result]
    return "\n".join(model["tgt"].DecodePieces(hyps))


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass  # quiet; the Go side logs

    def do_POST(self):
        q = parse_qs(urlparse(self.path).query)
        from_lang = q.get("from", ["en"])[0]
        to_lang = q.get("to", ["be"])[0]

        length = int(self.headers.get("Content-Length", 0))
        text = self.rfile.read(length).decode("utf-8", "replace") if length else ""

        try:
            body = translate(from_lang, to_lang, text).encode("utf-8")
            status = 200
        except Exception:
            traceback.print_exc()
            body = b"translation error"
            status = 500

        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    print(f"starting server on port {PORT}...", flush=True)
    ThreadingHTTPServer(("", PORT), Handler).serve_forever()
