# Copyright (C) 2025 Denis Forveille titou10.titou10@gmail.com
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.4

ARG DRIVER_VERSION 
ARG BUILD_DATE
ARG GIT_COMMIT 
ARG CSI_VERSION

ARG DEBIAN_FRONTEND=noninteractive  # Suppress debconf "unable to initialize frontend: Dialog" messages


LABEL org.opencontainers.image.title="CSI Driver for TrueNAS SCALE" \
      org.opencontainers.image.description="A CSI driver for dynamic provisioning of NFS volumes on TrueNAS SCALE." \
      org.opencontainers.image.authors="Denis Forveille <titou10.titou10@gmail.com>" \
      org.opencontainers.image.license="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/titou10titou10/csi-driver-truenas-scale" \
      org.opencontainers.image.version=${DRIVER_VERSION} \
      org.opencontainers.image.revision=${GIT_COMMIT} \
      org.opencontainers.image.created=${BUILD_DATE} \
      io.k8s.description="A CSI driver for TrueNAS SCALE with support for dynamic provisioning and snapshots." \
      io.k8s.display-name="CSI TrueNAS Scale Driver" \
      io.k8s.csi.version="${CSI_VERSION}" \
      io.k8s.csi.driver="tns.csi.titou10.org"
  

COPY /bin/tnsplugin /tnsplugin

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-mark unhold libcap2 && \
    clean-install ca-certificates mount nfs-common netbase

ENTRYPOINT ["/tnsplugin"]
