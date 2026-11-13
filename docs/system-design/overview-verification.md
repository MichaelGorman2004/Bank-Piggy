# Household summary fit

The At a glance card no longer scrolls internally. Income and balance tables keep their natural height and align at the top. Container queries use the allocated card size, rather than the browser width, to choose two columns, stacked tables, or a compact Income/Balances selector in the fixed footer.

Payroll estimates, the shared split, exact amounts, full member names, and balance notices remain available in Household details. The same fact readers supply the summary and dialog. Short cards abbreviate large positive or negative amounts while retaining the exact amount in the accessible label and title. Very short cards keep the full summary reachable through the existing details control instead of drawing facts over the footer. Other FinanceCard consumers retain their existing scroll behavior.

Overview's incomplete-plan notice now belongs to Plan totals. Its setup action is retained, and other planning views retain their existing notice. The household income prop is period-neutral because the Year view passes the yearly summary; the notice only checks for zero income.

## Verification

The root quality gate passed with contract/provider generation checks, formatting, source limits, lint, types, unused checks, Node tests, Go race tests with real PostgreSQL, and both builds. The existing nonfatal bundle-size warning remains. An independent read-only audit found no blocking issue; its income-prop naming finding was corrected.

Browser checks used an isolated loopback app and PostgreSQL schema with clearly labeled fictional accounts, placeholder credentials, and no provider access. No production account, mapping, credential, or financial record was changed during these checks.

- At 1512×823, 1366×801, 1366×900, 390×844, 320×568, 844×390, and 760×432, visible tables fit above the footer, the card had no internal scrolling, and the body had no horizontal overflow. Both compact table choices were checked.
- Long names and large positive/negative amounts fit. The details dialog retained the full names and exact amounts.
- Missing balances remained visible through Review balances. Empty manual balances and absent payroll showed explicit empty states.
- Enter opened the details dialog. Escape closed it and restored focus. The incomplete-plan action remained reachable in Plan totals and opened the existing editor.
- A stopped-server month request produced the real plan-load error and a 146-pixel card. The fallback fit, and all income and balance rows remained reachable in the dialog.
- A failed connection-status read retained the saved facts and exposed Reload balances. Restarting the isolated server and retrying cleared the error. Reload plan also recovered. A database comparison confirmed that the browser checks did not change the financial plan.

The review server, schema, temporary tab, and viewport override were removed. Screenshots and logs remain under `/tmp/bank-piggy-plaid-review`. These are desktop browser checks at mobile dimensions, not a physical-phone test. Production acceptance is a separate final step.
