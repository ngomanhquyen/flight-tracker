# Lab 1.4 — Solution

Thử làm trước khi đọc phần này.

> Bản thực thi: `bash scripts/solve.sh` chạy tất cả các phần, kèm tự kiểm
> tra `[PASS]`/`[FAIL]` — B2/C5/D2 (dựa trên `bot-service` thật) tự bỏ qua
> với cảnh báo nếu app chưa deploy. `bash scripts/cleanup.sh` dọn cả Pod
> tự tạo lẫn label/annotation đã thêm vào resource thật.

## Phần A — Tạo hàng loạt

```bash
for i in 1 2 3; do
  kubectl run pod-a$i --image=nginx:alpine --labels="env=dev,tier=web"
done

for i in 1 2 3; do
  kubectl run pod-b$i --image=nginx:alpine --labels="env=prod,tier=web"
done

kubectl run pod-c1 --image=nginx:alpine --labels="env=dev,tier=db"
```

## Phần B — Truy vấn bằng selector

### B1 — Pod tự tạo

```bash
# 1. env=dev
kubectl get pods -l env=dev

# 2. env in (dev, prod) — set-based, cú pháp có dấu ngoặc và khoảng trắng sau dấu phẩy tùy ý
kubectl get pods -l 'env in (dev,prod)'

# 3. tier=web AND env=dev — dấu phẩy giữa hai điều kiện = AND
kubectl get pods -l tier=web,env=dev

# 4. tier != db
kubectl get pods -l 'tier!=db'
```

**Về câu 4 — kết quả có thể khiến bạn bất ngờ nếu chạy trong namespace
`flight-tracker` (đã kiểm chứng thật): thay vì 6 pod (loại đúng `pod-c1`),
`tier!=db` khớp với **mọi** pod trong namespace không có `tier=db` — bao
gồm cả những pod hoàn toàn không mang label `tier` chút nào (`bot-service`,
`postgresql-0`, `sync-service-...`, Pod của lab 1.1/1.2 nếu còn tồn tại...).
Theo ngữ nghĩa selector chuẩn của Kubernetes, `key!=value` nghĩa là "label
đó **hoặc không tồn tại, hoặc tồn tại với giá trị khác**" — không phải "có
tồn tại và khác value". Muốn giới hạn đúng 6 pod web-tier (bắt buộc phải
*có* `tier` và khác `db`), phải kết hợp thêm điều kiện tồn tại:

```bash
# đúng 6 pod: có label tier, VÀ giá trị khác db
kubectl get pods -l tier,tier!=db
```

### B2 — Resource thật

```bash
# 5. Deployment + Service thuộc release flight-tracker, 2 loại resource 1 lệnh
kubectl get deployments,services -l app.kubernetes.io/part-of=flight-tracker

# 6. Y hệt selector đó nhưng target Pod
kubectl get pods -l app.kubernetes.io/part-of=flight-tracker
# No resources found — xem giải thích bên dưới
```

**Câu 6 — kết quả rỗng, đây là phát hiện chính của B2 (đã kiểm chứng
thật):**

```bash
kubectl get deployment bot-service -o jsonpath='{.metadata.labels}{"\n"}'
# {"app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"bot-service","app.kubernetes.io/part-of":"flight-tracker"}

kubectl get deployment bot-service -o jsonpath='{.spec.template.metadata.labels}{"\n"}'
# {"app.kubernetes.io/name":"bot-service"}
```

Chart này định nghĩa **hai** helper label khác nhau
([`_helpers.tpl`](../../../deployments/helm/flight-tracker/charts/common/templates/_helpers.tpl)):
`common.labels` (đủ cả `name`/`part-of`/`managed-by`, gắn vào
`metadata.labels` của Deployment/Service/ConfigMap/Secret — tức bản thân
các **object**) và `common.selectorLabels` (chỉ có `name`, gắn vào
`spec.template.metadata.labels` — tức **Pod template**, cũng là
`spec.selector.matchLabels` của Deployment). Pod được tạo ra từ
`spec.template`, nên nó chỉ thừa hưởng `name`, không bao giờ có
`part-of`/`managed-by` — bất kể Deployment "cha" của nó có gắn gì ở
`metadata.labels` riêng. Đây là thiết kế có chủ đích, không phải thiếu sót:
comment trong chart ghi rõ `common.selectorLabels` "must be a subset of
common.labels and immutable" — `spec.selector.matchLabels` của một
Deployment **không được sửa** sau khi tạo, nên giữ nó tối giản (chỉ 1 field)
tránh việc lỡ tay đổi `common.labels` (thêm/sửa `part-of`, `managed-by`...)
sau này làm vỡ selector.

```bash
# 7. Riêng bot-service (name thì Pod CÓ)
kubectl get pods -l app.kubernetes.io/name=bot-service

# 8. Một trong 3 service đã implement — set-based, 1 lệnh duy nhất
kubectl get pods -l 'app.kubernetes.io/name in (bot-service,flight-service,sync-service)'
```

**Câu 9:** Pod ở Phần A **không** xuất hiện trong kết quả câu 7/8, vì chúng
không hề mang label `app.kubernetes.io/name` — `kubectl run` không tự gắn
label nào ngoài những gì bạn truyền qua `--labels`, và bạn chỉ đặt
`env=`/`tier=`. Đây chính là lý do labs này có thể chạy an toàn trong cùng
namespace với app thật: hai bộ label (`tier=`/`env=` của lab, và
`app.kubernetes.io/*` của Helm chart) không giao nhau, nên không selector
nào ở lab vô tình khớp nhầm resource thật, và ngược lại.

## Phần C — Cập nhật label hàng loạt

### Trên Pod tự tạo

```bash
kubectl label pods -l tier=web owner=team-a
```

Đổi giá trị mà không có `--overwrite`:

```bash
kubectl label pods -l tier=web owner=team-b
# error: 'owner' already has a value (team-a), and --overwrite is false
```

Với `--overwrite`:

```bash
kubectl label pods -l tier=web owner=team-b --overwrite
```

**Vì sao có rào chắn này:** `kubectl label` mặc định coi việc ghi đè một
label đã tồn tại là hành động nguy hiểm (rất dễ gây mất label cũ hàng loạt
do gõ nhầm selector) — nên bắt buộc phải xác nhận rõ ràng bằng
`--overwrite`. Đặt một label **chưa từng có** thì không cần flag này.

Xoá label (dấu `-` ở cuối tên label, không có dấu cách trước `-`):

```bash
kubectl label pod pod-c1 tier-
```

### C5 — Trên Pod thật

```bash
POD=$(kubectl get pods -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[0].metadata.name}')
kubectl label pod "$POD" lab=ckad-1.4
kubectl get pod "$POD" --show-labels
```

**Câu hỏi "tồn tại mãi hay không":** Không. So sánh với nguồn "sự thật" mà
ReplicaSet của Deployment dùng để tạo Pod:

```bash
kubectl get deployment bot-service -o jsonpath='{.spec.template.metadata.labels}{"\n"}'
# chỉ có app.kubernetes.io/name: bot-service — không có "lab"
```

Label bạn vừa thêm chỉ tồn tại trên **đúng Pod đang chạy đó**, sửa trực
tiếp qua API, hoàn toàn nằm ngoài tầm biết của Deployment. Lần tới
Deployment tạo Pod mới thay thế Pod này (rolling update do đổi image/env,
Pod bị evict, node restart...), ReplicaSet dùng lại
`spec.template.metadata.labels` để dựng Pod — không có `lab=ckad-1.4` —
nên label biến mất trên Pod mới. Đây là khác biệt cốt lõi giữa **sửa một
object đang chạy** và **sửa desired state của controller quản lý nó**: cái
trước là tạm thời và không tái lập được, cái sau mới là thay đổi "thật" và
bền vững — kỳ thi CKAD hay kiểm tra đúng sự phân biệt này (ví dụ: "sửa
image của Pod" vs "sửa image trong Deployment").

## Phần D — Annotation

### Trên Pod tự tạo

```bash
kubectl annotate pods -l env=prod \
  description="Production workload - do not delete"
```

Cú pháp `kubectl annotate` đối xứng với `kubectl label` gần như hoàn toàn,
kể cả việc cũng cần `--overwrite` khi annotation đã tồn tại và dấu `-` để
xoá.

### D2 — Trên Deployment thật

```bash
kubectl get pods -l app.kubernetes.io/name=bot-service   # ghi lại AGE trước

kubectl annotate deployment bot-service inspected-by="ckad-lab-1.4"

kubectl get pods -l app.kubernetes.io/name=bot-service   # AGE không đổi
kubectl describe deployment bot-service | grep -A2 Annotations
```

**Vì sao an toàn, không trigger rollout:** `kubectl annotate deployment
bot-service ...` (không có thêm cờ path nào khác) mặc định sửa
`metadata.annotations` **ở cấp Deployment object**, không đụng tới
`spec.template.metadata.annotations` (annotation của Pod template). Chỉ
thay đổi ở `spec.template.*` mới khiến ReplicaSet nhận ra "desired Pod spec
đã đổi" và tạo rolling update — annotation ở cấp ngoài chỉ là metadata mô
tả bản thân Deployment, không ảnh hưởng gì tới Pod nó quản lý. Đây cũng là
field kubectl CKAD hay đánh vào: nhầm giữa annotate/label một Deployment và
annotate/label pod template bên trong nó có thể vô tình gây ra một rollout
ngoài ý muốn trên hệ thống thật.

## Verify

```bash
kubectl get pods --show-labels
kubectl get pods -l owner=team-a
kubectl get pod pod-c1 --show-labels
kubectl describe pod pod-b1 | grep -A2 Annotations
```

## Dọn dẹp

```bash
# theo tên, không theo -l tier=web — Lab 1.1's speed-pod cũng mang label
# đó, dùng selector sẽ vô tình xoá luôn Pod của lab khác (đã kiểm chứng
# thật khi viết lab này)
kubectl delete pod pod-a1 pod-a2 pod-a3 pod-b1 pod-b2 pod-b3 pod-c1

# nếu đã làm C5/D2:
kubectl label pod "$POD" lab-
kubectl annotate deployment bot-service inspected-by-
```
