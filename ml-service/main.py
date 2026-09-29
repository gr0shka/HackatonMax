"""OpenRouter-assisted route optimizer with a deterministic offline fallback."""
import json
import logging
import math
import os
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), format="%(asctime)s [%(levelname)s] %(message)s", handlers=[logging.StreamHandler(sys.stdout)])
OPENROUTER_URL = os.getenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1").rstrip("/")
OPENROUTER_MODEL = os.getenv("OPENROUTER_MODEL", "openai/gpt-4o-mini")
OPENROUTER_TIMEOUT = float(os.getenv("OPENROUTER_TIMEOUT", "18"))


def distance_km(a, b):
    lat1, lon1, lat2, lon2 = map(math.radians, (a["lat"], a["lon"], b["lat"], b["lon"]))
    dlat, dlon = lat2 - lat1, lon2 - lon1
    value = math.sin(dlat / 2) ** 2 + math.cos(lat1) * math.cos(lat2) * math.sin(dlon / 2) ** 2
    return 6371 * 2 * math.asin(math.sqrt(value))


def merged_interests(profiles):
    totals, count = {}, max(1, len(profiles))
    for profile in profiles:
        for category, value in profile.get("interests", {}).items():
            totals[category.lower()] = totals.get(category.lower(), 0) + float(value)
    return {key: value / count for key, value in totals.items()}


def local_optimize(body):
    """Greedy fallback balancing interests, rating and route detour."""
    budget = max(15, int(body.get("budget_minutes", 120)))
    money_budget = max(0, int(body.get("budget_rub", 0)))
    candidates = body.get("candidate_places", [])
    interests = merged_interests(body.get("user_profiles", []))
    current = body.get("start", {"lat": 0, "lon": 0})
    finish = body.get("finish", current)
    remaining, selected, spent, money_spent = list(candidates), [], 0, 0

    def estimated_cost(place):
        category = str(place.get("category", "")).lower()
        if any(word in category for word in ("restaurant", "food", "кафе", "ресторан", "coffee", "кофе")):
            return 700
        if any(word in category for word in ("museum", "art", "музей", "театр", "gallery")):
            return 450
        if any(word in category for word in ("souvenir", "сувенир", "shop", "магазин")):
            return 600
        return 0
    while remaining and len(selected) < 5:
        def score(place):
            category = str(place.get("category", "")).lower()
            affinity = max((weight for key, weight in interests.items() if key in category or category in key), default=0.35)
            detour = distance_km(current, place) + distance_km(place, finish) - distance_km(current, finish)
            return affinity * 4 + float(place.get("rating", 0)) * 0.35 - detour * 0.5
        place = max(remaining, key=score)
        duration = max(15, min(90, int(place.get("avg_duration_min") or 30)))
        place_cost = estimated_cost(place)
        walking = round((distance_km(current, place) + distance_km(place, finish)) / 4.5 * 60)
        if spent + duration + walking > budget or (money_budget and money_spent + place_cost > money_budget):
            remaining.remove(place)
            continue
        selected.append({"place_id": str(place["id"]), "order": len(selected) + 1, "allocated_time_min": duration, "estimated_cost_rub": place_cost})
        spent += duration + round(distance_km(current, place) / 4.5 * 60)
        money_spent += place_cost
        current = place
        remaining.remove(place)
    if not selected and candidates:
        place = max(candidates, key=lambda item: float(item.get("rating", 0)))
        duration = max(10, min(max(10, budget - 10), int(place.get("avg_duration_min") or 20)))
        place_cost = estimated_cost(place)
        if not money_budget or place_cost <= money_budget:
            selected.append({"place_id": str(place["id"]), "order": 1, "allocated_time_min": duration, "estimated_cost_rub": place_cost})
        spent = duration
    return {"selected_places": selected, "total_estimated_minutes": min(budget, spent + 10), "match_score": round(min(0.92, 0.62 + 0.06 * len(selected)), 2), "match_reasons": ["Места подобраны по интересам и рейтингу", "Порядок точек уменьшает лишние переходы", "Маршрут укладывается в заданное время"], "optimizer": "local-fallback"}


def openrouter_optimize(body):
    api_key = os.getenv("OPENROUTER_API_KEY", "").strip()
    if not api_key:
        raise RuntimeError("OPENROUTER_API_KEY is not configured")
    system = ("Ты оптимизатор городских маршрутов. Выбирай только place_id из входного списка. "
              "Учитывай интересы группы, рейтинг, время, близость и разнообразие. Маршрут должен укладываться в budget_minutes. "
              "Учитывай budget_rub и transport_mode. Верни только JSON: selected_places[{place_id,order,allocated_time_min,estimated_cost_rub}], total_estimated_minutes, match_score (0..1), match_reasons (2-4 строки на русском).")
    payload = {"model": OPENROUTER_MODEL, "temperature": 0.2, "response_format": {"type": "json_object"}, "messages": [{"role": "system", "content": system}, {"role": "user", "content": json.dumps(body, ensure_ascii=False)}]}
    request = Request(f"{OPENROUTER_URL}/chat/completions", data=json.dumps(payload).encode(), headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json", "HTTP-Referer": os.getenv("OPENROUTER_SITE_URL", "http://localhost:3000"), "X-Title": os.getenv("OPENROUTER_APP_NAME", "Go Route Planner")}, method="POST")
    with urlopen(request, timeout=OPENROUTER_TIMEOUT) as response:
        provider_data = json.loads(response.read().decode())
    return json.loads(provider_data["choices"][0]["message"]["content"])


def validate_result(result, body):
    allowed = {str(place["id"]): place for place in body.get("candidate_places", [])}
    budget, money_budget = max(15, int(body.get("budget_minutes", 120))), max(0, int(body.get("budget_rub", 0)))
    valid, used, money_used, seen = [], 0, 0, set()
    for item in result.get("selected_places", []):
        place_id = str(item.get("place_id", ""))
        if place_id not in allowed or place_id in seen:
            continue
        duration = max(10, min(90, int(item.get("allocated_time_min") or 20)))
        cost = max(0, int(item.get("estimated_cost_rub") or 0))
        if used + duration > budget or (money_budget and money_used + cost > money_budget):
            continue
        seen.add(place_id); used += duration; money_used += cost
        valid.append({"place_id": place_id, "order": len(valid) + 1, "allocated_time_min": duration, "estimated_cost_rub": cost})
        if len(valid) == 5: break
    if not valid: raise ValueError("model returned no valid candidates")
    raw_reasons = result.get("match_reasons", [])
    if isinstance(raw_reasons, str):
        raw_reasons = [raw_reasons]
    reasons = [str(x)[:180] for x in raw_reasons if str(x).strip()][:4]
    return {"selected_places": valid, "total_estimated_minutes": min(budget, max(used, int(result.get("total_estimated_minutes") or used))), "match_score": max(0.0, min(1.0, float(result.get("match_score", .75)))), "match_reasons": reasons or ["Маршрут оптимизирован с учетом интересов группы"], "optimizer": "openrouter", "model": OPENROUTER_MODEL}


def optimize(body):
    try:
        return validate_result(openrouter_optimize(body), body)
    except (RuntimeError, ValueError, KeyError, TypeError, json.JSONDecodeError, HTTPError, URLError, TimeoutError) as error:
        logging.warning("OpenRouter unavailable, using local optimizer: %s", error)
        return local_optimize(body)


class MLHandler(BaseHTTPRequestHandler):
    def send_json(self, status, payload):
        data = json.dumps(payload, ensure_ascii=False).encode()
        self.send_response(status); self.send_header("Content-Type", "application/json; charset=utf-8"); self.send_header("Content-Length", str(len(data))); self.end_headers(); self.wfile.write(data)
    def do_GET(self):
        self.send_json(200, {"status": "ok", "service": "route-ai", "openrouter": bool(os.getenv("OPENROUTER_API_KEY")), "model": OPENROUTER_MODEL}) if self.path in ("/", "/health", "/api/v1/health") else self.send_json(404, {"error": "not found"})
    def do_POST(self):
        if self.path != "/api/v1/optimize": self.send_json(404, {"error": "not found"}); return
        try:
            size = int(self.headers.get("Content-Length", 0))
            if size <= 0 or size > 2_000_000: raise ValueError("request body is empty or too large")
            body = json.loads(self.rfile.read(size))
            if not body.get("candidate_places"): raise ValueError("candidate_places must not be empty")
            result = optimize(body); logging.info("Optimized %d candidates via %s", len(body["candidate_places"]), result.get("optimizer")); self.send_json(200, result)
        except (ValueError, json.JSONDecodeError) as error: self.send_json(400, {"error": str(error)})
        except Exception: logging.exception("Unexpected optimization error"); self.send_json(500, {"error": "optimization failed"})
    def log_message(self, message, *args): logging.info("%s - %s", self.address_string(), message % args)


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8001")); logging.info("Starting route AI on port %d (model=%s)", port, OPENROUTER_MODEL); ThreadingHTTPServer(("0.0.0.0", port), MLHandler).serve_forever()
