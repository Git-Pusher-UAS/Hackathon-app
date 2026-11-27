// Prueba de carga k6. La corre el job "load" del CI; en local:
//   k6 run -e BASE_URL=http://localhost:3001 -e LOAD_USUARIO=admin -e LOAD_PASSWORD=... tests/load/k6_load_test.js
//
// Falla (exit code != 0) si se rompe algún umbral. Con LOAD_USUARIO y LOAD_PASSWORD
// hace login una sola vez en setup() (el login tiene rate limit) y carga rutas
// autenticadas; un 401 cuenta como fallo, no como éxito.
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';
const PROTECTED_PATHS = ['/api/auth/me', '/api/v1/users/me'];

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '30s', target: 10 },
    { duration: '5s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
    checks: ['rate>0.99'],
  },
};

export function setup() {
  if (!__ENV.LOAD_USUARIO || !__ENV.LOAD_PASSWORD) return { token: null };

  const res = http.post(
    `${BASE_URL}/api/auth/login`,
    JSON.stringify({ usuario: __ENV.LOAD_USUARIO, password: __ENV.LOAD_PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  const token = res.status === 200 ? res.json('token') : null;
  if (!token) throw new Error(`login falló en setup(): HTTP ${res.status} ${res.body}`);
  return { token };
}

export default function (data) {
  const health = http.get(`${BASE_URL}/health`);
  check(health, { 'health 200': (r) => r.status === 200 });

  if (data.token) {
    for (const path of PROTECTED_PATHS) {
      const res = http.get(`${BASE_URL}${path}`, {
        headers: { Authorization: `Bearer ${data.token}` },
      });
      check(res, { [`${path} 200`]: (r) => r.status === 200 });
    }
  }

  sleep(1);
}
