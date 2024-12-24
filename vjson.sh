vcmd() {
	COMMAND=$(echo "$1" | sed 's/;\s*/;/g')
	shift
	START_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
	START_TIMESTAMP=$(date +%s)
	TOTAL_STEPS=0
	CURRENT_STEP=0
	STEP_OUTPUTS=()
	ERRORS=()
	if [[ "$COMMAND" == *";"* ]]; then
		IFS=';' read -ra STEPS <<< "$COMMAND"
		TOTAL_STEPS=${#STEPS[@]}
		for ((i=0; i<TOTAL_STEPS; i++)); do
			CURRENT_STEP=$((i + 1))
			STEP_CMD="${STEPS[$i]}"
			STEP_CMD=$(echo "$STEP_CMD" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
			STEP_START=$(date +%s)
			STEP_OUTPUT=$(eval "$STEP_CMD" 2>&1)
			STEP_STATUS=$?
			STEP_END=$(date +%s)
			STEP_ELAPSED=$((STEP_END - STEP_START))
			STEP_INFO=$(jq -n \
				--arg cmd "$STEP_CMD" \
				--arg output "$STEP_OUTPUT" \
				--arg status "$([[ $STEP_STATUS -eq 0 ]] && echo "success" || echo "error")" \
				--arg elapsed "${STEP_ELAPSED}s" \
				--arg step "$CURRENT_STEP" \
				--arg total "$TOTAL_STEPS" \
				'{
					command: $cmd,
					output: $output,
					status: $status,
					elapsed_time: $elapsed,
					step: $step,
					total: $total
				}')

			STEP_OUTPUTS+=("$STEP_INFO")
			if [ $STEP_STATUS -ne 0 ]; then
				ERRORS+=("Step $CURRENT_STEP failed: $STEP_OUTPUT")
				break
			fi
		done
	else
		TOTAL_STEPS=1
		CURRENT_STEP=1
		COMMAND=$(echo "$COMMAND" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
		STEP_OUTPUT=$(eval "$COMMAND" 2>&1)
		STEP_STATUS=$?
		[ $STEP_STATUS -ne 0 ] && ERRORS+=("Command failed: $STEP_OUTPUT")
		STEP_INFO=$(jq -n \
			--arg cmd "$COMMAND" \
			--arg output "$STEP_OUTPUT" \
			--arg status "$([[ $STEP_STATUS -eq 0 ]] && echo "success" || echo "error")" \
			--arg elapsed "0s" \
			--arg step "1" \
			--arg total "1" \
			'{
				command: $cmd,
				output: $output,
				status: $status,
				elapsed_time: $elapsed,
				step: $step,
				total: $total
			}')
		STEP_OUTPUTS+=("$STEP_INFO")
	fi
	END_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
	END_TIMESTAMP=$(date +%s)
	ELAPSED_TIME=$((END_TIMESTAMP - START_TIMESTAMP))
	STEPS_JSON=$(printf '%s\n' "${STEP_OUTPUTS[@]}" | jq -s '.')
	ERRORS_JSON=$(printf '%s\n' "${ERRORS[@]}" | jq -R . | jq -s .)
	jq -n \
		--arg status "$([[ ${#ERRORS[@]} -eq 0 ]] && echo "success" || echo "error")" \
		--arg command "$COMMAND" \
		--arg start_time "$START_TIME" \
		--arg end_time "$END_TIME" \
		--arg elapsed_time "${ELAPSED_TIME}s" \
		--argjson steps "$STEPS_JSON" \
		--argjson errors "$ERRORS_JSON" \
		--arg current_step "$CURRENT_STEP" \
		--arg total_steps "$TOTAL_STEPS" \
		'{
			status: $status,
			data: {
				command: $command,
				start_time: $start_time,
				end_time: $end_time,
				elapsed_time: $elapsed_time,
				progress: {
					current: ($current_step | tonumber),
					total: ($total_steps | tonumber),
					percentage: (($current_step | tonumber) * 100 / ($total_steps | tonumber))
				},
				steps: $steps,
				errors: $errors
			}
		}'
}
greet() { echo "Hello, $1!"; }
vcmd "echo Step 1;  echo Step 2;   echo Step 3; greet John"