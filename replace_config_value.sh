#!/bin/bash

if [ $# -ne 2 ]; then
  echo "Uso: $0 clave nuevo_valor"
  echo "Ejemplo: $0 ip_memory 192.168.0.32"
  exit 1
fi

CLAVE="$1"
NUEVO_VALOR="$2"

DIRS=(cpu io kernel memoria)

for dir in "${DIRS[@]}"; do
  CONFIG_PATH="$dir/configs"
  if [ -d "$CONFIG_PATH" ]; then
    for archivo in "$CONFIG_PATH"/*.json; do
      if [ -f "$archivo" ]; then
        echo "Modificando $archivo ..."
        jq --indent 4 --arg clave "$CLAVE" --arg valor "$NUEVO_VALOR" \
          'if has($clave) and (.[$clave]|type == "number") then .[$clave] = ($valor|tonumber) elif has($clave) then .[$clave] = $valor else . end' \
          "$archivo" > "${archivo}.tmp" && mv "${archivo}.tmp" "$archivo"
      fi
    done
  fi
done

echo "Modificación completada."
