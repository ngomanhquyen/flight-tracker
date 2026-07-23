# Lab 2.3 — Scale & HPA

**CKAD domain:** Application Deployment (20%)

Bài này dùng `bot-service` (Deployment thật trong namespace `flight-tracker`)
— chọn nó thay vì `flight-service` vì không cần chờ Postgres (`dbCheck`
initContainer), scale/xoá nhanh, không ảnh hưởng dữ liệu gì. Deploy local
hiện đang tắt autoscaling (`values-local.yaml: autoscaling.enabled: false`)
nên **chưa có HPA nào tồn tại** cho `bot-service` — lab này tạo HPA mới,
không đụng độ với Helm.

## 0. Chuẩn bị: cài metrics-server

```bash
kubectl config set-context --current --namespace=flight-tracker
kubectl top nodes
```

Trên Docker Desktop Kubernetes, lệnh này sẽ báo lỗi kiểu
`error: Metrics API not available` — cụm này **không cài sẵn
metrics-server**, mà HPA bắt buộc cần nó để đọc %CPU thật của từng Pod. Cài:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch deployment metrics-server -n kube-system --type='json' \
  -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl rollout status deployment/metrics-server -n kube-system
```

`--kubelet-insecure-tls` cần thiết vì kubelet của node Docker Desktop dùng
chứng chỉ tự ký mà metrics-server mặc định không tin (lỗi sẽ là
`x509: certificate signed by unknown authority` nếu thiếu cờ này) — đây là
đặc thù cluster dev cá nhân (Docker Desktop/kind/minikube), **không cần và
không nên** làm vậy trên 1 cluster thi thật/production (ở đó metrics-server
đã được cài sẵn với chứng chỉ hợp lệ).

```bash
kubectl top nodes
kubectl top pods -n flight-tracker
```

Cả 2 lệnh giờ phải ra số liệu thật (có thể mất 30–60s sau khi
metrics-server Ready mới có mẫu đầu tiên).

## 1. Scale thủ công lên 10

```bash
kubectl get deployment bot-service -o jsonpath='{.spec.replicas}{"\n"}'
# ghi lại số này — dùng để khôi phục ở bước Dọn dẹp
```

```bash
kubectl scale deployment bot-service --replicas=10 -n flight-tracker
kubectl get pods -l app.kubernetes.io/name=bot-service -w
# Ctrl+C khi đủ 10 pod Running
```

`kubectl scale` chỉ sửa đúng 1 field (`spec.replicas`) — imperative, có
hiệu lực ngay, không tạo hay cần HPA. Lưu ý: `strategy.rollingUpdate`
(`maxUnavailable: 0 / maxSurge: 1`) đã thấy ở lab 2.1 **không áp dụng** ở
đây — chiến lược rolling update chỉ chi phối lúc **thay đổi Pod template**
(image, env...), còn thao tác scale thuần tuý (thêm/bớt số bản sao giống
hệt nhau) thì Kubernetes tạo/xoá Pod trực tiếp, không qua cơ chế surge/
unavailable nào cả.

## 2. Cấu hình HPA ở 50% CPU

```bash
kubectl autoscale deployment bot-service --cpu-percent=50 --min=2 --max=6 -n flight-tracker
kubectl get hpa bot-service -n flight-tracker
```

`kubectl autoscale` là cách nhanh để tạo 1 `HorizontalPodAutoscaler` (tương
đương viết YAML `kind: HorizontalPodAutoscaler` với
`targetCPUUtilizationPercentage: 50`, `minReplicas: 2`, `maxReplicas: 6`).
Cột `TARGETS` (dạng `<current>%/<target>%`) có thể hiện `<unknown>/50%`
trong vài chục giây đầu — bình thường, đợi metrics-server có mẫu đầu tiên
cho Pod này.

> Bản `kubectl` mới có thể báo `--cpu-percent` deprecated, gợi ý dùng
> `--cpu=50%` thay thế — cả 2 vẫn chạy được, không phải lỗi.

**Điều kiện bắt buộc để HPA tính được %, và 1 gotcha thật đã gặp khi viết
lab này:** HPA tính %CPU cho **toàn bộ Pod**, nên **mọi container trong Pod**
đều phải khai `resources.requests.cpu` — chỉ cần **1 container thiếu** là
toàn bộ metric của Pod đó thành `<unknown>` mãi mãi, dù các container khác
đã khai đủ. `bot-service` (container chính, `100m`) và `log-shipper`
(sidecar log, `10m` — xem `common.logShipperContainer`) đã có sẵn, nhưng
Pod `bot-service` khi deploy local còn 2 sidecar nữa qua `extraContainers`
trong `values-local.yaml`: `cloudflared` và `webhook-registrar` — ban đầu
**không khai `resources` gì cả**. Chạy thử lab này lần đầu, `kubectl describe
hpa bot-service` báo thẳng nguyên nhân:

```
Warning  FailedGetResourceMetric  ...  failed to get cpu utilization:
  missing request for cpu in container cloudflared of Pod bot-service-...
```

Đã sửa cả `values-local.yaml` và `values-local.yaml.example` để thêm
`resources.requests: { cpu: 100m, memory: 100Mi }` cho 2 sidecar này (giá
trị tuỳ ý, vì chúng gần như không có tải thật), và thêm `resources.requests`
tương tự cho initContainer `wait-for-db` (`common.initContainers` —
`flight-service`/`sync-service` sẽ gặp đúng lỗi này nếu bạn thử làm lại lab
tương tự trên 2 service đó). Nếu bạn tự thêm sidecar/initContainer nào khác
sau này, luôn nhớ khai `resources.requests.cpu` cho nó — thiếu 1 container
là đủ để HPA "mù" hoàn toàn với cả Pod.

## 3. Quan sát HPA "thắng" manual scale

```bash
kubectl get hpa bot-service -n flight-tracker -w
```

Theo dõi cột `REPLICAS`. Vì `maxReplicas: 6` < 10 (số bạn set tay ở bước 1),
HPA sẽ **ép ngay lập tức** replica count về trong khoảng `[2, 6]` ở lần
đồng bộ đầu tiên (mặc định mỗi 15s) — việc này xảy ra tức thời, không phụ
thuộc %CPU đang đo được bao nhiêu, vì `min`/`max` là giới hạn cứng.

Sau khi vào trong khoảng `[2, 6]`, HPA mới bắt đầu điều chỉnh tiếp **dựa
trên %CPU thật** — vì `bot-service` gần như rảnh (idle, gần 0% CPU so với
target 50%), nó sẽ tiếp tục giảm dần về đúng `minReplicas: 2`. Điểm cần lưu
ý: bước giảm này **có thể mất tới 5 phút** mới ổn định hẳn ở 2 — HPA (từ
`autoscaling/v2`) có "stabilization window" mặc định 300s cho quyết định
**scale-down** để tránh dao động liên tục (scale lên/xuống lặp lại khi tải
dao động ngắn hạn); scale-**up** thì không có độ trễ này, gần như tức thời
ở lần đồng bộ tiếp theo. Nếu sau vài giây bạn thấy `REPLICAS` vẫn còn 10 rồi
mới xuống 6 rồi mới xuống 2 (không nhảy thẳng về 2), đó là hành vi đúng, không
phải lab chạy sai.

Ctrl+C khi thấy ổn định (thường là `REPLICAS=2`).

## 4. Dọn dẹp

```bash
kubectl delete hpa bot-service -n flight-tracker
kubectl scale deployment bot-service --replicas=<số đã ghi ở bước 1> -n flight-tracker
```

Hoặc chạy lại Helm để đồng bộ hết mọi field theo `values.yaml`/
`values-local.yaml` (bao gồm cả replica count):

```bash
cd deployments/helm/flight-tracker && ./deploy-local.sh
```

`metrics-server` có thể **giữ lại** — nó vô hại, hữu ích cho `kubectl top`
và các lab HPA sau này. Muốn gỡ hẳn:

```bash
kubectl delete -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

## Điểm cần hiểu, không chỉ chép lệnh

- **`kubectl scale` chỉ có hiệu lực tới lần đồng bộ HPA kế tiếp** — một khi
  đã có HPA gắn vào Deployment, **HPA mới là chủ sở hữu thật sự** của
  `spec.replicas`; số bạn set tay sẽ bị ghi đè ngay khi không nằm trong
  `[min, max]`, và tiếp tục bị điều chỉnh theo metric sau đó.
- **`min`/`max` là giới hạn cứng, áp dụng ngay**; %CPU chỉ quyết định giá
  trị **trong** khoảng đó, không quyết định việc có vượt khoảng hay không.
- **HPA cần 2 điều kiện tiên quyết, thiếu 1 trong 2 đều ra `<unknown>`
  mãi mãi:** metrics-server đã cài + chạy được, và **mọi container trong
  Pod** (không riêng container chính) có khai `resources.requests.cpu` —
  đã gặp thật ở lab này: `cloudflared`/`webhook-registrar` (sidecar thêm qua
  `extraContainers`) thiếu `resources` khiến cả Pod báo `<unknown>` dù
  `bot-service` chính đã khai đủ.
- **Scale-up gần như tức thời, scale-down có "stabilization window"
  (mặc định 300s)** — thiết kế có chủ đích để tránh dao động liên tục khi
  tải dao động ngắn hạn; đừng nhầm là lab/HPA bị lỗi nếu số replica giảm
  chậm hơn hẳn so với lúc tăng.
- **Chiến lược rolling update (`maxUnavailable`/`maxSurge`) không áp dụng
  cho thao tác scale thuần tuý** — nó chỉ chi phối khi Pod template thay
  đổi (xem lại lab 2.1); scale chỉ thêm/bớt bản sao giống hệt nhau.
