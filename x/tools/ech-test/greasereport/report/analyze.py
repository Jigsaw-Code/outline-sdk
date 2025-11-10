import pandas as pd

df = pd.read_csv('grease-results-top1000.csv')

grouped = df.groupby('domain')

failure_diffs = []
performance_improvements = []
performance_degradations = []
total_domains = 0

for domain, group in grouped:
    total_domains += 1
    control_rows = group[group['ech_grease'] == False]
    grease_rows = group[group['ech_grease'] == True]
    if len(control_rows) > 0 and len(grease_rows) > 0:
        control_row = control_rows.iloc[0]
        grease_row = grease_rows.iloc[0]
        
        status_diff = control_row['http_status'] != grease_row['http_status']
        error_diff = str(control_row['curl_error_name']) != str(grease_row['curl_error_name'])
        
        if status_diff or error_diff:
            failure_diffs.append({
                "domain": domain,
                "control_status": control_row['http_status'],
                "control_error": control_row['curl_error_name'],
                "grease_status": grease_row['http_status'],
                "grease_error": grease_row['curl_error_name'],
            })
        else:
            time_diff = control_row['tls_handshake_ms'] - grease_row['tls_handshake_ms']
            if abs(time_diff) > 100:
                if time_diff > 0:
                    performance_improvements.append({
                        "domain": domain,
                        "control_ms": control_row['tls_handshake_ms'],
                        "grease_ms": grease_row['tls_handshake_ms'],
                        "diff_ms": time_diff
                    })
                else:
                    performance_degradations.append({
                        "domain": domain,
                        "control_ms": control_row['tls_handshake_ms'],
                        "grease_ms": grease_row['tls_handshake_ms'],
                        "diff_ms": time_diff
                    })

print('--- Analysis of ECH GREASE Test ---')

print(f'\nTotal domains tested: {total_domains}')

print('\n--- Failure Differences ---')
if failure_diffs:
    print(f'Found {len(failure_diffs)} domains with different success/failure outcomes:')
    for item in failure_diffs:
        print(f"  Domain: {item['domain']}")
        print(f"    Control: status={item['control_status']}, error='{item['control_error']}'")
        print(f"    Grease:  status={item['grease_status']}, error='{item['grease_error']}'")
else:
    print('No domains with failure differences found.')

print('\n--- Performance Differences (TLS Handshake Time) ---')
if performance_improvements or performance_degradations:
    print(f'Found {len(performance_improvements) + len(performance_degradations)} domains with significant (>100ms) performance differences.')
    
    print(f'\n  Improved performance (faster with GREASE): {len(performance_improvements)} domains')
    for item in performance_improvements:
        print(f"    - {item['domain']}: {item['diff_ms']}ms faster")

    print(f'\n  Degraded performance (slower with GREASE): {len(performance_degradations)} domains')
    for item in performance_degradations:
        print(f"    - {item['domain']}: {-item['diff_ms']}ms slower")
else:
    print('No significant performance differences found.')

print('\n--- Summary ---')
print(f'Total domains with any difference: {len(failure_diffs) + len(performance_improvements) + len(performance_degradations)}')