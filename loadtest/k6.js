import http from "k6/http";
import { check } from "k6";

const baseURL = __ENV.BASE_URL || "http://localhost:8080";
const token = __ENV.JWT_TOKEN || "";

export const options = {
  stages: [
    { duration: "30s", target: 100 },
    { duration: "1m", target: 1000 },
    { duration: "1m", target: 2500 },
    { duration: "2m", target: 5000 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(99)<1000"],
  },
};

export default function () {
  const response = http.get(`${baseURL}/api/users/42`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  check(response, {
    "gateway returned a successful response": (res) => res.status === 200,
  });
}
