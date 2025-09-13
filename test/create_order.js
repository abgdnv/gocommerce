import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '3m',
};

const accessToken = 'YOUR_TOKEN';
const productID = 'YOUR_PRODUCT_ID';

export default function () {
  const url = 'http://api_gateway:8080/api/orders';
  const payload = JSON.stringify({
    "user_id": "YOUR_USER_ID",
    "status": "pending",
    "items": [
        {
        "product_id": productID,
        "quantity": 1,
        "price_per_item": 100,
        "price": 100
        }
    ]
});

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${accessToken}`,
    },
  };

  const res = http.post(url, payload, params);
  check(res, { 'status was 201 or 5xx': (r) => r.status === 201 || r.status >= 500 });
  sleep(1);
}
