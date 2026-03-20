ARG WORKDIR=/src
ARG CRDS_DIR=${WORKDIR}/crds
ARG PROGRAM=oiax

FROM golang:alpine AS deps
RUN apk add --no-cache git github-cli


FROM deps AS download
ARG WORKDIR
WORKDIR ${WORKDIR}
COPY go.mod go.sum ./
RUN --mount=type=secret,id=github_token,env=GH_APP_TOKEN \
    echo "${GH_APP_TOKEN}" | gh auth login --with-token \
    && gh auth setup-git \
    && go mod download


FROM download AS crd
ARG CRDS_DIR
WORKDIR ${CRDS_DIR}
RUN cp $(go list -m -f '{{.Dir}}' github.com/katastroma/tropis)/config/crd/*.yaml .


FROM download AS src
ARG CRDS_DIR
COPY --link . .
COPY --link --from=crd ${CRDS_DIR}/*.yaml cmd/init/crds/


FROM src AS prep
ARG CGO_ENABLED=0
ARG PROGRAM
RUN go build -v ./...


FROM prep AS build
RUN go build -v -ldflags="-w -s" -o ${PROGRAM} cmd/main/main.go


FROM scratch AS runtime
ARG WORKDIR
ARG PROGRAM
ENTRYPOINT [ "/program" ]


FROM runtime
COPY --from=build-main ${WORKDIR}/${PROGRAM} /program
