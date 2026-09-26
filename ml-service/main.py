import json
import logging
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)

class MLHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path in ("/health", "/api/v1/health", "/"):
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok", "service": "ml-service-stub"}).encode("utf-8"))
            return

        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path == "/api/v1/optimize":
            content_length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(content_length)) if content_length > 0 else {}

            budget_minutes = body.get("budget_minutes", 120)
            candidates = body.get("candidate_places", [])
            user_profiles = body.get("user_profiles", [])
            logging.info(
                "Received optimization request: candidates=%d, users=%d, budget=%d min",
                len(candidates),
                len(user_profiles),
                budget_minutes,
            )

            # Select up to 4 candidates that fit within the time budget
            selected = []
            accumulated_time = 0
            for place in candidates:
                duration = place.get("avg_duration_min", 20)
                if duration <= 0:
                    duration = 20
                if accumulated_time + duration > budget_minutes and len(selected) > 0:
                    break
                selected.append({
                    "place_id": place["id"],
                    "order": len(selected) + 1,
                    "allocated_time_min": duration,
                })
                accumulated_time += duration
                if len(selected) >= 5:
                    break

            resp = {
                "selected_places": selected,
                "total_estimated_minutes": accumulated_time + 20,
                "match_score": 0.94,
                "match_reasons": [
                    "Оптимальный баланс интересов группы (культура, гастрономия, отдых)",
                    "Маршрут укладывается в доступный бюджет времени",
                    "Высокий средний рейтинг выбранных локаций",
                ],
            }

            resp_bytes = json.dumps(resp, ensure_ascii=False).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(resp_bytes)))
            self.end_headers()
            self.wfile.write(resp_bytes)
            return

        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        logging.info("%s - %s" % (self.address_string(), format % args))

if __name__ == "__main__":
    port = 8001
    logging.info("Starting ML Mock Service on port %d...", port)
    server = HTTPServer(("0.0.0.0", port), MLHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
