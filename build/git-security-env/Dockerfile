# syntax = docker/dockerfile:1.4.0

FROM alpine

RUN <<EOF cat > /env.sh
#!/bin/sh
echo \$GIT_REPO
EOF

RUN <<EOF cat > /user_defined_env.sh
#!/bin/sh
echo \$USER_DEFINED_ENV
EOF

RUN chmod 755 /*.sh
