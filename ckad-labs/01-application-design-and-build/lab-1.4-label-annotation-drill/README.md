# Lab 1.4 — Label & Annotation Drill

**Thời lượng:** ~30 phút | **CKAD domain:** Application Design and Build (20%)

## Mục tiêu

- Tạo hàng loạt Pod và cập nhật label trên nhiều object cùng lúc.
- Truy vấn resource bằng label selector (equality và set-based) — trên cả
  Pod tự tạo lẫn Pod thật của `bot-service`/`flight-service`/`sync-service`.
- Hiểu rõ khi nào `kubectl label` cần `--overwrite`.
- Phân biệt sửa label/annotation trực tiếp trên một object đang chạy với
  sửa desired state trong một controller (Deployment) — và vì sao hai việc
  đó có hệ quả khác nhau hẳn.

## Chuẩn bị

Dùng lại namespace `flight-tracker`.

Phần B2/C5/D2 dùng resource thật của project — cần `bot-service` (và lý
tưởng là `flight-service`, `sync-service`) đã deploy (`./deploy-local.sh`,
xem Lab 1.2 Phần A). Không có thì bỏ qua các bước đó, làm phần còn lại vẫn
đủ trọn vẹn kỹ năng.

## Nhiệm vụ

### Phần A — Tạo hàng loạt

`kubectl` không có lệnh "tạo nhiều object cùng lúc" — dùng vòng lặp shell.
Tạo 7 Pod, image `nginx:alpine`, như sau:

- `pod-a1`, `pod-a2`, `pod-a3` → labels `env=dev,tier=web`
- `pod-b1`, `pod-b2`, `pod-b3` → labels `env=prod,tier=web`
- `pod-c1` → labels `env=dev,tier=db`

### Phần B — Truy vấn bằng selector

**B1 — trên Pod tự tạo.** Viết lệnh `kubectl get pods` cho từng yêu cầu sau
(không dùng `grep`, chỉ dùng `-l`/`--selector`):

1. Tất cả pod có `env=dev`.
2. Tất cả pod có `env` là `dev` **hoặc** `prod` (set-based).
3. Tất cả pod có `tier=web` **và** `env=dev` (AND, hai điều kiện cùng lúc).
4. Tất cả pod **không** có `tier=db`.

**B2 — trên resource thật.** Chart này dùng bộ nhãn chuẩn
`app.kubernetes.io/*` (xem
[`_helpers.tpl`](../../../deployments/helm/flight-tracker/charts/common/templates/_helpers.tpl)),
nhưng **không đều** giữa các loại object — đây chính là điều bài này cho
bạn tự phát hiện, hoàn toàn tách biệt với `tier=`/`env=` bạn vừa dùng ở B1.

5. Tất cả Deployment **và** Service thuộc release `flight-tracker` — một
   selector `app.kubernetes.io/part-of=flight-tracker` chạy trên cả 2 loại
   resource cùng lúc (`kubectl get deployments,services -l ...`).
6. Chạy lại **đúng selector đó** nhưng đổi mục tiêu sang Pod
   (`kubectl get pods -l app.kubernetes.io/part-of=flight-tracker`). Kết
   quả có gì khác câu 5 không? Dự đoán trước, rồi tự tra nguyên nhân bằng
   cách so `metadata.labels` của Deployment `bot-service` với
   `spec.template.metadata.labels` của chính nó
   (`kubectl get deployment bot-service -o jsonpath=...` cho cả 2 path).
7. Riêng Pod của `bot-service` (`app.kubernetes.io/name` — label này thì
   Pod **có**).
8. Pod thuộc **một trong ba** service đã có code thật —
   `bot-service`, `flight-service`, `sync-service` — bằng **một** selector
   set-based duy nhất trên `app.kubernetes.io/name`, không phải 3 lệnh
   riêng.
9. So sánh: Pod bạn tạo ở Phần A có nằm trong kết quả câu 7/8 không? Vì sao
   (hoặc vì sao không)?

### Phần C — Cập nhật label hàng loạt

**Trên Pod tự tạo:**

10. Gắn thêm `owner=team-a` cho **tất cả** pod có `tier=web` — dùng một lệnh
    `kubectl label` với selector, không lặp qua từng pod.
11. Sau đó thử đổi `owner` của cùng nhóm pod đó thành `team-b` **mà không**
    thêm `--overwrite`. Quan sát lỗi. Rồi chạy lại có `--overwrite`.
12. Xoá hẳn label `tier` khỏi `pod-c1` (cú pháp dấu `-` ở cuối tên label).

**C5 — trên Pod thật (an toàn, không ảnh hưởng ứng dụng đang chạy):**

13. Gắn label `lab=ckad-1.4` trực tiếp lên **một** Pod thật của
    `bot-service` (không phải lên Deployment). Xác nhận label xuất hiện.
    Sau đó tự trả lời: label này có tồn tại mãi không, hay sẽ biến mất ở
    lần Pod bị tạo lại tiếp theo (rollout mới, hoặc Pod bị evict)? Vì sao?
    (Gợi ý: so sánh với nơi Deployment lấy `spec.template.metadata.labels`
    để tạo Pod — `kubectl get deployment bot-service -o jsonpath=...`.)

### Phần D — Annotation

**Trên Pod tự tạo:**

14. Thêm annotation `description="Production workload - do not delete"` cho
    toàn bộ pod có `env=prod`.

**D2 — trên resource thật (an toàn — xem giải thích trong `solution.md`
trước khi chạy nếu bạn còn phân vân):**

15. Thêm annotation `inspected-by="ckad-lab-1.4"` lên **Deployment**
    `bot-service` (không phải lên Pod, không phải vào `spec.template`).
    Xác nhận nó **không** làm Pod của `bot-service` bị restart
    (`kubectl get pods -l app.kubernetes.io/name=bot-service` — so `AGE`
    trước/sau).

## Tiêu chí hoàn thành

- [ ] `kubectl get pods --show-labels` → đủ 7 pod với label đúng như Phần A
- [ ] 4 lệnh selector ở B1 trả về đúng tập pod mong đợi (lưu ý câu 4 —
      `tier!=db` khớp cả những pod khác trong namespace không có label
      `tier` chút nào, không chỉ 6 pod bạn vừa tạo; xem giải thích ở
      `solution.md` nếu số đếm làm bạn bất ngờ)
- [ ] Câu 5 (Deployment/Service) và câu 6 (Pod, cùng selector) ở B2 cho ra
      kết quả **khác nhau**, và bạn giải thích được vì sao
- [ ] Lệnh đổi `owner` không có `--overwrite` **phải báo lỗi** trước khi bạn
      thêm flag đó
- [ ] `kubectl get pod pod-c1 --show-labels` → không còn label `tier`
- [ ] `kubectl describe pod pod-b1` → phần `Annotations` có dòng `description`
- [ ] (nếu làm C5/D2) Bạn trả lời đúng cả 2 câu hỏi "vì sao" — xem đối
      chiếu ở `solution.md`
- [ ] (nếu làm D2) `AGE` của các Pod `bot-service` không đổi trước/sau bước 15

## Dọn dẹp

```bash
kubectl delete pod pod-a1 pod-a2 pod-a3 pod-b1 pod-b2 pod-b3 pod-c1
```

(Xoá theo tên, không dùng `-l tier=web` — Lab 1.1's `speed-pod` cũng mang
label đó, selector sẽ vô tình xoá luôn Pod của lab khác nếu nó còn tồn
tại.)

Nếu đã làm C5/D2, dọn luôn label/annotation bạn thêm vào resource thật
(dấu `-` ở cuối để xoá):

```bash
kubectl label pod <tên-pod-bot-service> lab-
kubectl annotate deployment bot-service inspected-by-
```
