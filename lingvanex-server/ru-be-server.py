import sentencepiece as spm
from ctranslate2 import Translator
from http.server import BaseHTTPRequestHandler, HTTPServer

source = "ru"
target = "be"
path_to_model = f"./{source}_{target}/1"

translator = Translator(path_to_model, compute_type="int8")
source_tokenizer = spm.SentencePieceProcessor(f"{path_to_model}/{source}.spm.model")
target_tokenizer = spm.SentencePieceProcessor(f"{path_to_model}/{target}.spm.model")

text = ["Ну что, начинаем?!"]

input_tokens = source_tokenizer.EncodeAsPieces(text)
translator_output = translator.translate_batch(
    input_tokens,
    batch_type="tokens",
    beam_size=2,
    max_input_length=0,
    max_decoding_length=256,
)

output_tokens = [item.hypotheses[0] for item in translator_output]
translation = target_tokenizer.DecodePieces(output_tokens)
print("\n".join(translation))


class SimpleGetHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        # Get content length to read the body
        content_length = int(self.headers.get("Content-Length", 0))
        post_data = self.rfile.read(content_length)

        text = post_data.splitlines()

        input_tokens = source_tokenizer.EncodeAsPieces(text)
        translator_output = translator.translate_batch(
            input_tokens,
            batch_type="tokens",
            beam_size=2,
            max_input_length=0,
            max_decoding_length=256,
        )

        output_tokens = [item.hypotheses[0] for item in translator_output]
        translation = target_tokenizer.DecodePieces(output_tokens)
        print("\n".join(translation))

        # Set response status code
        self.send_response(200)
        # Set headers
        self.send_header("Content-type", "text/plain; charset=utf-8")
        self.end_headers()
        # Write response body
        self.wfile.write("\n".join(translation).encode("utf-8"))


port = 8000
server_address = ("", port)
httpd = HTTPServer(server_address, SimpleGetHandler)
print(f"Starting server on port {port}...")
httpd.serve_forever()
