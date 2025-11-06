import pandas as pd

df = pd.read_csv('/Users/fortuna/firehook/outline-sdk/x/tools/ech-test/workspace/grease-results-top1000.csv')

grouped = df.groupby('domain')

print('Domains with differences between control and grease runs (TLS handshake time):')
print('---')
domains_with_diffs = 0
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
        time_diff = abs(control_row['tls_handshake_ms'] - grease_row['tls_handshake_ms']) > 100

        if status_diff or error_diff or time_diff:
            domains_with_diffs += 1
            print(f"Domain: {domain}")
            print(f"  Control: http_status={control_row['http_status']}, error='{control_row['curl_error_name']}', tls_handshake_ms={control_row['tls_handshake_ms']}")
            print(f"  Grease:  http_status={grease_row['http_status']}, error='{grease_row['curl_error_name']}', tls_handshake_ms={grease_row['tls_handshake_ms']}")
            print('---')

print(f'\nSummary: Found differences in {domains_with_diffs} out of {total_domains} domains.')
