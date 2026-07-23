# Lab 3.3 — ServiceAccount & RBAC

**Thời lượng:** ~60 phút | **CKAD domain:** Application Environment, Configuration and Security (25%)

## Mục tiêu

- Tạo ServiceAccount + Role + RoleBinding, giới hạn đúng 1 quyền cụ thể
  (`list`/`get`/`watch` Pod trong 1 namespace).
- Từ 1 Pod dùng ServiceAccount đó, tự gọi Kubernetes API bằng `curl` +
  token được tự động mount sẵn (không dùng `kubectl` bên trong Pod) để
  list Pod — và list ra được đúng các Pod **thật** của project (`bot-service`,
  `flight-service`, `postgresql`...).
- Đối chiếu với thực tế: hiện **không service nào** trong project gọi
  Kubernetes API — tất cả đều dùng ServiceAccount `default` ngầm định
  (không có `serviceAccountName` nào được set trong
  `common.deployment`/`common.cronjob`), và ServiceAccount `default` không
  có RBAC nào gắn thêm ngoài quyền mặc định của cluster. Nên bài này là bài
  tập **tự chứa hoàn toàn**, không có resource thật nào để đối chiếu ngoài
  việc list đúng Pod thật đang chạy.

## Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

## Nhiệm vụ

1. Tạo ServiceAccount `pod-lister`:

   ```bash
   kubectl create serviceaccount pod-lister
   ```

2. Tạo Role `pod-reader` — chỉ cho phép `get`/`list`/`watch` trên `pods`,
   chỉ trong namespace `flight-tracker` (dùng `Role`, không phải
   `ClusterRole` — phạm vi phải giới hạn đúng 1 namespace):

   ```bash
   kubectl create role pod-reader \
     --verb=get --verb=list --verb=watch --resource=pods
   ```

3. Tạo RoleBinding gắn `pod-reader` cho `pod-lister`:

   ```bash
   kubectl create rolebinding pod-reader-binding \
     --role=pod-reader --serviceaccount=flight-tracker:pod-lister
   ```

4. Viết `manifests/api-caller-pod.yaml`: Pod tên `api-caller-pod`, image
   `curlimages/curl:8.11.0`, `spec.serviceAccountName: pod-lister`, lệnh
   `sleep 3600`.

5. Apply, rồi `kubectl exec` vào Pod, tự gọi Kubernetes API bằng token +
   CA cert được Kubernetes tự động mount sẵn tại
   `/var/run/secrets/kubernetes.io/serviceaccount/`:

   ```bash
   kubectl exec -it api-caller-pod -- sh -c '
     TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
     curl -s --cacert /var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
       -H "Authorization: Bearer $TOKEN" \
       https://kubernetes.default.svc/api/v1/namespaces/flight-tracker/pods
   '
   ```

6. So sánh: gọi thử đúng API đó nhưng xin 1 hành động **không** được Role
   cho phép — ví dụ `DELETE` 1 Pod, hoặc list Pod ở namespace khác
   (`kube-system`) — phải bị từ chối `403 Forbidden`.

## Tiêu chí hoàn thành

- [ ] Response JSON của bước 5 liệt kê đúng các Pod thật đang chạy trong `flight-tracker`
- [ ] Gọi thử hành động không được Role cho phép → HTTP `403`
- [ ] `kubectl auth can-i list pods --as=system:serviceaccount:flight-tracker:pod-lister -n flight-tracker` → `yes`
- [ ] `kubectl auth can-i delete pods --as=system:serviceaccount:flight-tracker:pod-lister -n flight-tracker` → `no`
- [ ] `kubectl auth can-i list pods --as=system:serviceaccount:flight-tracker:pod-lister -n kube-system` → `no`

## Dọn dẹp

```bash
kubectl delete pod api-caller-pod --ignore-not-found
kubectl delete rolebinding pod-reader-binding --ignore-not-found
kubectl delete role pod-reader --ignore-not-found
kubectl delete serviceaccount pod-lister --ignore-not-found
```
