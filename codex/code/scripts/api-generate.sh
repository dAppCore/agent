#!/bin/bash

# Default values
output_format="ts"
routes_file="routes/api.php"
output_file="api_client" # Default output file name without extension

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        generate) ;; # Skip the generate subcommand
        --ts) output_format="ts";;
        --js) output_format="js";;
        --openapi) output_format="openapi";;
        *) routes_file="$1";;
    esac
    shift
done

# Set the output file extension based on format
if [[ "$output_format" == "openapi" ]]; then
    output_file="openapi.json"
else
    output_file="api_client.${output_format}"
fi

# Function to parse the routes file
parse_routes() {
    if [ ! -f "$1" ]; then
        echo "Error: Routes file not found at $1" >&2
        exit 1
    fi
    awk -F"'" '
    /Route::apiResource/ {
        resource = $2;
        resource_singular = resource;
        sub(/s$/, "", resource_singular);
        print "GET " resource " list";
        print "POST " resource " create";
        print "GET " resource "/{" resource_singular "} get";
        print "PUT " resource "/{" resource_singular "} update";
        print "DELETE " resource "/{" resource_singular "} delete";
    }
    /Route::(get|post|put|delete|patch)/ {
        line = $0;
        match(line, /Route::([a-z]+)/, m);
        method = toupper(m[1]);
        uri = $2;
        action = $6;
        print method " " uri " " action;
    }
    ' "$1"
}

# Function to generate the API client
generate_client() {
    local format=$1
    local outfile=$2
    local client_object="export const api = {\n"
    local dto_definitions=""
    declare -A dtos

    declare -A groups

    # First pass: Collect all routes and DTOs
    while read -r method uri action; do
        group=$(echo "$uri" | cut -d'/' -f1)
        if [[ -z "${groups[$group]}" ]]; then
            groups[$group]=""
        fi
        groups[$group]+="$method $uri $action\n"

        if [[ "$method" == "POST" || "$method" == "PUT" || "$method" == "PATCH" ]]; then
            local resource_name_for_dto=$(echo "$group" | sed 's/s$//' | awk '{print toupper(substr($0,0,1))substr($0,2)}')
            local dto_name="$(tr '[:lower:]' '[:upper:]' <<< ${action:0:1})${action:1}${resource_name_for_dto}Dto"
            dtos[$dto_name]=1
        fi
    done

    # Generate DTO interface definitions for TypeScript
    if [ "$format" == "ts" ]; then
        for dto in $(echo "${!dtos[@]}" | tr ' ' '\n' | sort); do
            dto_definitions+="export interface ${dto} {}\n"
        done
        dto_definitions+="\n"
    fi

    # Sort the group names alphabetically to ensure consistent output
    sorted_groups=$(for group in "${!groups[@]}"; do echo "$group"; done | sort)

    for group in $sorted_groups; do
        client_object+="  ${group}: {\n"

        # Sort the lines within the group by the action name (field 3)
        sorted_lines=$(echo -e "${groups[$group]}" | sed '/^$/d' | sort -k3)

        while IFS= read -r line; do
            if [ -z "$line" ]; then continue; fi
            method=$(echo "$line" | cut -d' ' -f1)
            uri=$(echo "$line" | cut -d' ' -f2)
            action=$(echo "$line" | cut -d' ' -f3)

            params=$(echo "$uri" | grep -o '{[^}]*}' | sed 's/[{}]//g')
            ts_types=""
            js_args=""

            # Generate arguments for the function signature
            for p in $params; do
                js_args+="${p}, "
                ts_types+="${p}: number, "
            done

            # Add a 'data' argument for POST/PUT/PATCH methods
            if [[ "$method" == "POST" || "$method" == "PUT" || "$method" == "PATCH" ]]; then
                local resource_name_for_dto=$(echo "$group" | sed 's/s$//' | awk '{print toupper(substr($0,0,1))substr($0,2)}')
                local dto_name="$(tr '[:lower:]' '[:upper:]' <<< ${action:0:1})${action:1}${resource_name_for_dto}Dto"
                ts_types+="data: ${dto_name}"
                js_args+="data"
            fi

            # Clean up function arguments string
            func_args=$(echo "$ts_types" | sed 's/,\s*$//' | sed 's/,$//')
            js_args=$(echo "$js_args" | sed 's/,\s*$//' | sed 's/,$//')

            final_args=$([ "$format" == "ts" ] && echo "$func_args" || echo "$js_args")

            # Construct the fetch call string
            fetch_uri="/api/${uri}"
            fetch_uri=$(echo "$fetch_uri" | sed 's/{/${/g')

            client_object+="    ${action}: (${final_args}) => fetch(\`${fetch_uri}\`"

            # Add request options for non-GET methods
            if [ "$method" != "GET" ]; then
                client_object+=", {\n      method: '${method}'"
                if [[ "$method" == "POST" || "$method" == "PUT" || "$method" == "PATCH" ]]; then
                    client_object+=", \n      body: JSON.stringify(data)"
                fi
                client_object+="\n    }"
            fi
            client_object+="),\n"

        done <<< "$sorted_lines"
        client_object+="  },\n"
    done

    client_object+="};"

    echo -e "// Generated from ${routes_file}\n" > "$outfile"
    echo -e "${dto_definitions}${client_object}" >> "$outfile"
    echo "API client generated at ${outfile}"
}

# Function to generate OpenAPI spec
generate_openapi() {
    local outfile=$1
    local paths_json=""

    declare -A paths
    while read -r method uri action; do
        path="/api/${uri}"
        # OpenAPI uses lowercase methods
        method_lower=$(echo "$method" | tr '[:upper:]' '[:lower:]')

        # Group operations by path
        if [[ -z "${paths[$path]}" ]]; then
            paths[$path]=""
        fi
        paths[$path]+="\"${method_lower}\": {\"summary\": \"${action}\"},"
    done

    # Assemble the paths object
    sorted_paths=$(for path in "${!paths[@]}"; do echo "$path"; done | sort)
    for path in $sorted_paths; do
        operations=$(echo "${paths[$path]}" | sed 's/,$//') # remove trailing comma
        paths_json+="\"${path}\": {${operations}},"
    done
    paths_json=$(echo "$paths_json" | sed 's/,$//') # remove final trailing comma

    # Create the final OpenAPI JSON structure
    openapi_spec=$(cat <<EOF
{
  "openapi": "3.0.0",
  "info": {
    "title": "API Client",
    "version": "1.0.0",
    "description": "Generated from ${routes_file}"
  },
  "paths": {
    ${paths_json}
  }
}
EOF
)

    echo "$openapi_spec" > "$outfile"
    echo "OpenAPI spec generated at ${outfile}"
}


# Main logic
parsed_routes=$(parse_routes "$routes_file")

if [[ "$output_format" == "ts" || "$output_format" == "js" ]]; then
    generate_client "$output_format" "$output_file" <<< "$parsed_routes"
elif [[ "$output_format" == "openapi" ]]; then
    generate_openapi "$output_file" <<< "$parsed_routes"
else
    echo "Invalid output format specified." >&2
    exit 1
fi
