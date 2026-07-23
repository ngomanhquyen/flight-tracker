# Lab 2.1 — Rolling Update & Rollback

**CKAD domain:** Application Deployment (20%)

Bài này thực hành trên `flight-service` — Deployment **thật** trong namespace
`flight-tracker` (không tạo resource giả). `flight-service` do Helm quản lý
(`deploy-local.sh`), nên mọi lệnh `kubectl set image`/`kubectl edit` bên dưới
chỉ tồn tại tới lần `helm upgrade` kế tiếp — giống lưu ý đã gặp ở lab 1.2/1.3
với `bot-service`/`sync-service`. Đây là hành vi bình thường, không phải lỗi:
Helm luôn là nguồn sự thật cuối cùng (`values.yaml`/`values-local.yaml`).

Từng lệnh dưới đây đều có giải thích **tại sao**, không chỉ chép để chạy.

## 0. Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
kubectl get deployment flight-service
```

Set namespace mặc định để khỏi phải gõ `-n flight-tracker` mỗi lệnh (các
lệnh dưới vẫn giữ `-n flight-tracker` tường minh, phòng khi bạn chạy lẻ từng
lệnh mà chưa set context). `kubectl get deployment flight-service` xác nhận
Deployment đang tồn tại và đang chạy — nếu chưa deploy, chạy
`./deploy-local.sh` từ `deployments/helm/flight-tracker` trước.

## 1. Chuẩn bị 2 "phiên bản" image

```bash
docker tag flight-tracker/flight-service:local flight-tracker/flight-service:v1
docker tag flight-tracker/flight-service:local flight-tracker/flight-service:v2
```

Cả `v1` và `v2` trỏ **cùng một image content** (chỉ khác tag) — bài này luyện
*cơ chế* rolling update (Deployment nhận biết thay đổi image → tạo ReplicaSet
mới → chuyển dần traffic), không phải luyện build code thật. Không cần
`docker build` lại: `docker tag` là đủ để có 2 tag khác nhau cho
`kubectl set image` phân biệt.

## 2. Rolling update: v1 → v2

```bash
kubectl set image deployment/flight-service \
  flight-service=flight-tracker/flight-service:v2 -n flight-tracker
```

`kubectl set image` sửa đúng 1 trường (`spec.template.spec.containers[].image`)
mà không cần `kubectl edit` (mở editor, dễ gõ sai) hay viết lại cả file YAML
bằng `kubectl apply -f`. Cú pháp `<container-name>=<image>` bắt buộc nêu tên
container (`flight-service` — khớp tên container trong `common.deployment`)
vì 1 Pod có thể có nhiều container.

## 3. Theo dõi rollout

```bash
kubectl rollout status deployment/flight-service -n flight-tracker
```

Lệnh này **block** cho tới khi rollout xong hoặc lỗi — đúng cách CKAD hay
hỏi "theo dõi rollout" chứ không phải chỉ `kubectl get pods` một lần. Rollout
"thành công" nghĩa là: đủ số replica mới ở trạng thái Ready, ReplicaSet cũ đã
scale về 0.

Xem cơ chế bên dưới rõ hơn:

```bash
kubectl get rs -l app.kubernetes.io/name=flight-service -n flight-tracker
```

Trong lúc rollout đang chạy, lệnh này cho thấy **2 ReplicaSet cùng tồn tại**
một lúc (cũ đang scale down, mới đang scale up) — đây chính là ý nghĩa của
`strategy.rollingUpdate` mà `common.deployment` (trong
[`_helpers.tpl`](../../../deployments/helm/flight-tracker/charts/common/templates/_helpers.tpl))
đã cấu hình sẵn cho `flight-service`:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 0
    maxSurge: 1
```

- `maxSurge: 1` — cho phép tạo **thêm** tối đa 1 pod mới vượt quá
  `replicaCount` trong lúc rollout (không phải thay thế ngay pod cũ).
- `maxUnavailable: 0` — **không được phép** có pod nào "biến mất" khỏi số
  lượng sẵn sàng tại bất kỳ thời điểm nào trong rollout.

Kết hợp lại: Kubernetes luôn tạo pod mới **trước**, đợi nó Ready, rồi mới xoá
1 pod cũ — không bao giờ giảm số pod đang phục vụ xuống dưới mức desired.
Đây là cấu hình **zero-downtime rolling update** kinh điển, đánh đổi bằng
việc rollout chậm hơn (từng pod một) và cần dư tài nguyên cho pod "surge".

## 4. Giả lập một bản deploy lỗi

```bash
kubectl set image deployment/flight-service \
  flight-service=flight-tracker/flight-service:v3-broken -n flight-tracker
```

`v3-broken` là tag **chưa từng được build/tag** trên máy. Vì
`imagePullPolicy: Never` (xem `values-local.yaml`) — Kubernetes sẽ không thử
pull từ registry, và vì image không có sẵn local, pod mới sẽ vào trạng thái
`ErrImagePull` → `ImagePullBackOff`. Đây là cách giả lập "bản deploy lỗi" gọn
nhất, không cần build một binary thật sự bị crash.

```bash
kubectl rollout status deployment/flight-service -n flight-tracker --timeout=30s
```

Lệnh này sẽ **timeout/báo lỗi** sau 30s (rollout không bao giờ hoàn tất vì
pod mới không lên được) — đúng như mong đợi, không phải bug của bạn.

```bash
kubectl get pods -l app.kubernetes.io/name=flight-service -n flight-tracker
```

Quan sát: 1 pod mới `ImagePullBackOff`, nhưng (các) pod **cũ vẫn `Running`**.
Đây chính là hệ quả trực tiếp của `maxUnavailable: 0` ở bước 3 — Kubernetes
tuyệt đối không xoá pod cũ khi chưa xác nhận pod mới Ready. Kết quả: một bản
deploy lỗi khiến rollout bị **kẹt** (stuck), chứ không khiến app **down** —
đây là điểm CKAD hay hỏi để phân biệt "stuck rollout" với "outage".

## 5. Rollback

```bash
kubectl rollout undo deployment/flight-service -n flight-tracker
kubectl rollout status deployment/flight-service -n flight-tracker
```

`rollout undo` không chỉ đổi lại field `image` — nó phục hồi **toàn bộ**
`spec.template` (Pod template) của ReplicaSet trước đó (revision liền trước,
hoặc chỉ định bằng `--to-revision=N`). Vì ReplicaSet cũ (image `v2`, đang
`Running`) vẫn còn tồn tại (Kubernetes giữ lại một số ReplicaSet cũ theo
`revisionHistoryLimit`), rollback ở đây thực chất là quay lại đúng
ReplicaSet đó — nên diễn ra nhanh và cũng tuân theo `maxUnavailable`/
`maxSurge` y hệt một rollout bình thường.

## 6. Xem lịch sử rollout

```bash
kubectl rollout history deployment/flight-service -n flight-tracker
kubectl rollout history deployment/flight-service -n flight-tracker --revision=2
```

Mặc định cột `CHANGE-CAUSE` sẽ trống, vì Kubernetes **không tự động** ghi lại
lý do thay đổi — chỉ ghi khi có annotation `kubernetes.io/change-cause` trên
Pod template tại thời điểm đổi (cờ `--record` từng làm việc này tự động
nhưng đã bị deprecated). Muốn revision sau này có ghi chú rõ ràng:

```bash
kubectl annotate deployment/flight-service -n flight-tracker \
  kubernetes.io/change-cause="rollout v2 for lab 2.1" --overwrite
```

Annotate **trước khi** tạo thay đổi tiếp theo — annotation áp dụng cho
revision được tạo *sau* nó, không hồi tố cho revision đã có.

## Dọn dẹp

Khôi phục lại đúng trạng thái Helm quản lý (không để Deployment lệch khỏi
`values-local.yaml`):

```bash
kubectl set image deployment/flight-service \
  flight-service=flight-tracker/flight-service:local -n flight-tracker
kubectl rollout status deployment/flight-service -n flight-tracker
```

Hoặc — cách đúng đắn hơn về lâu dài — chạy lại Helm để nó tự đồng bộ mọi
field (không chỉ image) về đúng `values.yaml`/`values-local.yaml`:

```bash
cd deployments/helm/flight-tracker && ./deploy-local.sh
```

Nên ưu tiên cách thứ hai nếu bạn định làm tiếp lab khác ngay sau — sửa tay
bằng `kubectl set image` chỉ nên dùng để luyện phản xạ thao tác, không phải
cách quản lý Deployment lâu dài của project này.

## Điểm cần hiểu, không chỉ chép lệnh

- **Rolling update vs Recreate:** `RollingUpdate` (dùng ở đây) thay thế pod
  dần dần, giữ traffic liên tục; `Recreate` xoá hết pod cũ rồi mới tạo pod
  mới (có downtime, nhưng đơn giản hơn và cần thiết khi 2 phiên bản không
  thể chạy song song, ví dụ đổi schema DB không tương thích ngược).
- **`maxUnavailable`/`maxSurge` không phải chỉ là con số tuỳ ý** — cặp
  `(0, 1)` mà `flight-service` dùng nghĩa là "ưu tiên tuyệt đối uptime, chấp
  nhận rollout chậm hơn và tốn thêm tài nguyên tạm thời". Một cặp khác như
  `(1, 0)` sẽ ưu tiên ngược lại (không dùng thêm tài nguyên, chấp nhận giảm
  tạm capacity).
- **`rollout undo` phục hồi cả Pod template, không riêng gì `image`** — nếu
  bản deploy lỗi đổi cả image lẫn env/resources, undo sẽ phục hồi toàn bộ,
  không chỉ đổi lại image.
- **Một bản deploy lỗi không tự động = outage** — outcome cụ thể (stuck vs
  down) phụ thuộc hoàn toàn vào `maxUnavailable`. Đây là lý do CKAD/thực tế
  luôn khuyến nghị `maxUnavailable: 0` cho service quan trọng.
- **Helm vs kubectl trực tiếp:** mọi thao tác `kubectl set image`/`edit` ở
  trên là "sửa tay" desired state đang chạy trong cluster — không sửa
  `values.yaml`. Lần `helm upgrade` kế tiếp (kể cả không đổi gì trong
  `values.yaml`) sẽ ghi đè lại đúng những gì Helm nghĩ là đúng, xoá hết dấu
  vết của lab này. Đây là lý do bước Dọn dẹp ở trên tồn tại.
