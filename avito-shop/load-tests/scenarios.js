import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Пользовательские метрики
const purchaseFailRate = new Rate('purchase_fails');
const transferFailRate = new Rate('transfer_fails');
const authFailRate = new Rate('auth_fails');

// Тренды для отслеживания времени ответа
const purchaseLatency = new Trend('purchase_latency');
const transferLatency = new Trend('transfer_latency');
const authLatency = new Trend('auth_latency');

// Конфигурация тестов
export const options = {
  scenarios: {
    // Тест авторизации
    auth_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },  // Разогрев
        { duration: '1m', target: 50 },   // Тест
        { duration: '30s', target: 0 },   // Снижение нагрузки
      ],
      gracefulRampDown: '30s',
      exec: 'authScenario',
    },
    // Тест покупок
    purchase_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 30 },
        { duration: '1m', target: 30 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'purchaseScenario',
    },
    // Тест передачи монет
    transfer_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 40 },
        { duration: '1m', target: 40 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'transferScenario',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% запросов должны быть быстрее 500ms
    'purchase_fails': ['rate<0.1'],      // Не более 10% неудачных покупок
    'transfer_fails': ['rate<0.1'],      // Не более 10% неудачных переводов
    'auth_fails': ['rate<0.05'],         // Не более 5% неудачных авторизаций
  },
};

// Базовая конфигурация
const BASE_URL = 'http://localhost:8080/api';
const headers = {
  'Content-Type': 'application/json',
};

// Сценарий авторизации
export function authScenario() {
  const username = `user_${randomString(8)}`;
  const password = 'password123';

  const loginPayload = JSON.stringify({
    username: username,
    password: password,
  });

  const startTime = new Date();
  const loginRes = http.post(`${BASE_URL}/auth`, loginPayload, { headers });
  authLatency.add(new Date() - startTime);

  const success = check(loginRes, {
    'auth status is 200': (r) => r.status === 200,
    'has token': (r) => JSON.parse(r.body).token !== undefined,
  });

  if (!success) {
    authFailRate.add(1);
  } else {
    authFailRate.add(0);
  }

  sleep(1);
}

// Сценарий покупки
export function purchaseScenario() {
  // Сначала авторизуемся
  const username = `user_${randomString(8)}`;
  const loginRes = http.post(`${BASE_URL}/auth`, JSON.stringify({
    username: username,
    password: 'password123',
  }), { headers });

  const token = JSON.parse(loginRes.body).token;
  headers['Authorization'] = `Bearer ${token}`;

  // Список доступных товаров
  const items = ['t-shirt', 'cup', 'pen', 'book'];
  const randomItem = items[Math.floor(Math.random() * items.length)];

  const startTime = new Date();
  const purchaseRes = http.get(`${BASE_URL}/buy/${randomItem}`, { headers });
  purchaseLatency.add(new Date() - startTime);

  const success = check(purchaseRes, {
    'purchase status is 200': (r) => r.status === 200,
    'purchase successful': (r) => JSON.parse(r.body).message === 'purchase successful',
  });

  if (!success) {
    purchaseFailRate.add(1);
  } else {
    purchaseFailRate.add(0);
  }

  sleep(1);
}

// Сценарий передачи монет
export function transferScenario() {
  // Создаем двух пользователей
  const sender = `sender_${randomString(8)}`;
  const receiver = `receiver_${randomString(8)}`;

  // Авторизуем отправителя
  const loginRes = http.post(`${BASE_URL}/auth`, JSON.stringify({
    username: sender,
    password: 'password123',
  }), { headers });

  const token = JSON.parse(loginRes.body).token;
  headers['Authorization'] = `Bearer ${token}`;

  // Регистрируем получателя (тк он не существует)
    const loginResReceiver = http.post(`${BASE_URL}/auth`, JSON.stringify({
      username: receiver,
      password: 'password123',
    }), { headers });
  
    // const tokenReceiver = JSON.parse(loginResReceiver.body).token;
    // headers['Authorization'] = `Bearer ${tokenReceiver}`;

  // Отправляем монеты
  const transferPayload = JSON.stringify({
    toUser: receiver,
    amount: 100,
  });

  const startTime = new Date();
  const transferRes = http.post(`${BASE_URL}/sendCoin`, transferPayload, { headers });
  transferLatency.add(new Date() - startTime);

  const success = check(transferRes, {
    'transfer status is 200': (r) => {
      return r.status === 200
    },
    'transfer successful': (r) => {
        try {

            return JSON.parse(r.body).message === 'transfer successful';
        } catch (e) {
            console.log(`Failed to parse response: ${r.body}`);
            return false;
        }
    },
});

  if (!success) {
    transferFailRate.add(1);
  } else {
    transferFailRate.add(0);
  }

  sleep(1);
}