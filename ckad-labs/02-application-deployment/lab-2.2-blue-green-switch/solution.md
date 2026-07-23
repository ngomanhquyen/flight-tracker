# Lab 2.2 — Blue/Green Switch (qua Service selector của `flight-service`)

**CKAD domain:** Application Deployment (20%)

Bài gốc CKAD mô tả blue/green bằng 2 Deployment song song + 1 Service flip
`selector` giữa 2 label set. Ở đây mình luyện đúng **cơ chế lõi** đó (Service
`selector` quyết định traffic đi đâu, đổi được bất cứ lúc nào, không cần đợi
rollout) nhưng áp lên **resource thật**: Service `flight-service` trong
namespace `flight-tracker` — thay vì tạo 2 Deployment giả, bài này cố tình
**gãy selector** để nó không khớp Pod nào (giống hệt hiệu ứng khi bạn "flip"
sang 1 màu không tồn tại), quan sát hậu quả thật lên `bot-service`, rồi sửa
lại.

## Vì sao chọn `flight-service`, không phải `bot-service`

`cloudflared` (sidecar lo việc lộ `bot-service` ra Internet cho Telegram)
gọi thẳng `http://localhost:8080` — cùng Pod với `bot-service`, dùng chung
network namespace, **không đi qua Service nào cả**. Phá `selector` của
Service `bot-service` sẽ không ảnh hưởng gì tới việc bot nhận tin nhắn
Telegram thật.

Ngược lại, `bot-service` gọi sang `flight-service` bằng đúng tên DNS của
Service (`BOT_CLIENTS_FLIGHT_SERVICE_URL: "http://flight-service:8080"` —
xem `values.yaml` của `bot-service`) — đây là 2 Pod khác nhau, **bắt buộc**
phải qua Service để kube-proxy route đúng. Phá `selector` của Service
`flight-service` mới thật sự cắt được 1 tính năng thật của bot.

## 0. Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
kubectl get svc flight-service -o jsonpath='{.spec.selector}{"\n"}'
kubectl get endpoints flight-service
```

`kubectl get endpoints` liệt kê IP:port thật của (các) Pod đang khớp
`selector` — đây là cách nhanh nhất để chẩn đoán "Service tồn tại nhưng
không có gì đứng sau nó", một kỹ năng troubleshooting rất hay gặp trong đề
thi CKAD lẫn thực tế. Ở trạng thái bình thường, `selector` sẽ là
`{"app.kubernetes.io/name":"flight-service"}` và endpoints sẽ có ít nhất 1
địa chỉ IP Pod thật.

## 1. Xác nhận hoạt động bình thường (baseline)

```bash
kubectl run curl-test --rm -it --restart=Never \
  --image=curlimages/curl:8.11.0 -- curl -s -o /dev/null -w '%{http_code}\n' \
  http://flight-service:8080/health
```

Chạy 1 Pod tạm (`--rm` tự xoá sau khi thoát) gọi qua đúng tên DNS Service
(`flight-service`, không phải IP Pod) — kỳ vọng in ra `200`. Đây là baseline
để so sánh sau khi phá.

> Nếu bạn đang chạy `bot-service` thật với `cloudflared`/webhook thật, đây
> cũng là lúc thử gửi 1 lệnh tra cứu chuyến bay qua Telegram thật để có
> baseline "trước khi phá" — không bắt buộc, phần dưới vẫn quan sát được đầy
> đủ chỉ bằng `curl-test`.

## 2. Phá selector

```bash
kubectl patch svc flight-service -n flight-tracker -p '{"spec":{"selector":{"app.kubernetes.io/name":"flight-service-none"}}}'
```

`kubectl patch` chỉ merge đúng field được nêu (strategic merge patch), không
thay cả object như `kubectl apply -f`/`replace`. Giá trị
`flight-service-none` được chọn vì chắc chắn không Pod nào mang label này —
mô phỏng đúng hiệu ứng "Service trỏ sang 1 phiên bản/màu không tồn tại".

Điểm đáng chú ý: khác với `spec.selector` của **Deployment** (immutable —
khoá cứng ngay từ lúc tạo, không sửa được), `spec.selector` của **Service**
sửa được bất kỳ lúc nào, không giới hạn — đây chính là cơ chế mà blue/green
thật sự dùng để "flip" traffic tức thời giữa 2 tập Pod.

## 3. Quan sát hậu quả

```bash
kubectl get endpoints flight-service -n flight-tracker
```

Cột `ENDPOINTS` giờ là `<none>` — Service vẫn tồn tại (vẫn có ClusterIP),
nhưng không còn Pod nào đứng sau nó.

```bash
kubectl run curl-test --rm -it --restart=Never \
  --image=curlimages/curl:8.11.0 -- curl -s -m 5 -o /dev/null -w '%{http_code}\n' \
  http://flight-service:8080/health
```

Kết quả: **"Connection refused" gần như ngay lập tức**, không phải bị treo
chờ timeout. Đây là hành vi có chủ đích của kube-proxy: khi 1 Service
ClusterIP có 0 endpoint, nó lập tức reject kết nối tới VIP đó thay vì để
request treo vô định — giúp client (ở đây là `bot-service`) fail nhanh thay
vì đợi lâu.

**Hiệu ứng thật lên bot:** nếu lúc này bạn gõ lệnh tra cứu chuyến bay qua
Telegram thật, bot **vẫn phản hồi** (không bị treo/crash) nhưng trả về lỗi
thân thiện dạng "không kết nối được flight-service" — theo đúng thiết kế
graceful degradation của `bot-service` (đã nói ở lab 1.2:
`services/bot-service/README.md` — bot không có init container chờ
dependency, chấp nhận chạy ở chế độ hạn chế khi 1 dependency down, thay vì
bị k8s treo Pod ở `Init:`). Các lệnh **không** cần `flight-service` (ví dụ
`/start`, các lệnh liên quan `subscription-service`) vẫn hoạt động bình
thường suốt thời gian này.

## 4. Sửa lại

```bash
kubectl patch svc flight-service -n flight-tracker -p '{"spec":{"selector":{"app.kubernetes.io/name":"flight-service"}}}'
kubectl get endpoints flight-service -n flight-tracker
```

`ENDPOINTS` có lại IP Pod ngay lập tức — không có độ trễ lan truyền đáng kể,
vì kube-proxy watch trực tiếp Service/EndpointSlice qua API server.

```bash
kubectl run curl-test --rm -it --restart=Never \
  --image=curlimages/curl:8.11.0 -- curl -s -o /dev/null -w '%{http_code}\n' \
  http://flight-service:8080/health
```

Kỳ vọng lại `200` — và nếu bạn có test qua Telegram thật ở bước 1, thử lại
đúng lệnh đó để xác nhận bot hoạt động bình thường trở lại.

## Dọn dẹp

Service `flight-service` do Helm quản lý (`common.service` trong chart) —
`selector` đã được patch về đúng giá trị gốc ở bước 4 nên không cần thêm gì.
Nếu muốn chắc chắn 100% (hoặc lỡ quên bước 4), chạy lại Helm để nó tự đồng
bộ mọi field về đúng `values.yaml`:

```bash
cd deployments/helm/flight-tracker && ./deploy-local.sh
```

## Điểm cần hiểu, không chỉ chép lệnh

- **`spec.selector` của Service khác hẳn `spec.selector` của Deployment** —
  Deployment khoá cứng ngay từ lúc tạo (immutable, quyết định Deployment
  "sở hữu" Pod nào mãi mãi); Service thì hoàn toàn có thể đổi bất kỳ lúc
  nào, không giới hạn. Đây chính là cơ chế duy nhất mà 1 traffic switch
  blue/green thật sự cần.
- **`kubectl get endpoints`/`EndpointSlice` là công cụ chẩn đoán số 1** khi
  "Service tồn tại nhưng không hoạt động" — luôn kiểm tra endpoints trước
  khi nghi ngờ network policy, DNS, hay code, vì nguyên nhân phổ biến nhất
  chỉ là selector không khớp label Pod.
- **Service với 0 endpoint → Connection refused ngay, không phải timeout**
  — hành vi này giúp phân biệt "dependency down nhưng biết ngay" với "mạng
  bị treo" khi debug thật.
- **Không phải mọi lệnh gọi nội bộ đều qua Service** — so sánh với lab
  trước đó: `cloudflared → bot-service` là gọi `localhost` trong cùng Pod
  (không qua Service, phá selector không ảnh hưởng); `bot-service →
  flight-service` là gọi qua DNS Service thật (2 Pod khác nhau, bắt buộc
  qua Service). Luôn xác định rõ 2 Pod có **cùng Pod hay khác Pod** trước
  khi đoán một selector/NetworkPolicy có ảnh hưởng gì không.
- **Graceful degradation vs hard dependency** — `bot-service` chọn "chạy
  hạn chế khi thiếu dependency" (degrade 1 tính năng, không chết cả bot);
  `sync-service`/`flight-service` lại có `wait-for-db` initContainer (chặn
  cứng, không chạy nếu thiếu Postgres) vì với chúng, thiếu DB thì không thể
  chạy đúng ở bất kỳ mức nào. Chọn hướng nào tuỳ việc thiếu dependency đó có
  làm container chính chạy sai hoàn toàn hay chỉ mất 1 phần chức năng.
