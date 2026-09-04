const API_BASE = window.DIX_BASE || ""; // 由 template.html 内联注入(服务端替换 basePath)
        const apiUrl = (path) => (API_BASE ? API_BASE + path : path);

        function app() {
            return {
                // State
                loading: true,
                sidebarCollapsed: false,
                stats: { provider_count: 0, object_count: 0, package_count: 0, edge_count: 0 },
                packages: [],
                uiVersion: 'trace-entry-v5-20260323',
                packageSearch: '',
                globalSearch: '',
                searchResults: [],
                currentPackage: null,
                currentView: 'providers',
                currentLayout: 'hierarchical',
                currentDepth: '2',
                allData: null,
                selectedNode: null,
                network: null,
                focusedType: null,
                focusedGraph: null,
                focusedGroup: null,
                aggregateGroups: true,
                debugGroupMatching: false,
                groupRules: [],
                newGroupName: '',
                newGroupPrefix: '',
                storageKey: 'dix.groupRules.v1',
                groupMembers: {},
                filterPrefix: '',
                mermaidOpen: false,
                mermaidSource: '',
                mermaidSvg: '',
                mermaidError: '',
                lastGraphData: null,
                expandedGroups: [],
                runtimeStats: [],
                runtimeStatsLoading: false,
                runtimeStatsError: '',
                runtimeStatsSearch: '',
                runtimeStatsUpdatedAt: '',
                runtimeStatsOnlyExecuted: false,
                runtimeStatsCollapsed: true,
                recentErrors: [],
                recentErrorsLoading: false,
                recentErrorsError: '',
                recentErrorsUpdatedAt: '',
                recentErrorsCollapsed: true,
                traceRecords: [],
                traceLoading: false,
                traceError: '',
                traceUpdatedAt: '',
                traceLimit: 500,
                traceOperation: '',
                traceStatus: '',
                traceEvent: '',
                traceTreeMode: true,
                traceIndentMode: 'log',
                traceViewMode: 'span',
                traceOnlyErrorSpan: false,
                traceDepth: '0',
                traceCollapsedSpanKeys: {},
                traceCollapsedGroupKeys: {},
                traceFullscreen: false,
                errorTypeGuideCollapsed: true,
                errorTypeGuide: [
                    {
                        code: 'provider_registration_invalid',
                        summary: 'Provider 注册参数非法（如 nil、非函数、返回值不合法）。',
                        action: '检查 Provide/TryProvide 的入参签名与返回值数量。',
                    },
                    {
                        code: 'inject_dependency_missing',
                        summary: '注入阶段缺失依赖，当前类型没有可用 provider。',
                        action: '确认依赖已注册、类型精确匹配，并检查导入顺序。',
                    },
                    {
                        code: 'provider_input_unresolved',
                        summary: 'Provider 的输入参数无法解析（上游依赖缺失或不匹配）。',
                        action: '根据 input_type/input_types 逐级回溯缺失节点。',
                    },
                    {
                        code: 'provider_return_error',
                        summary: 'Provider 正常执行但返回了业务错误。',
                        action: '优先排查外部资源（DB/缓存/HTTP）与配置有效性。',
                    },
                    {
                        code: 'provider_panic',
                        summary: 'Provider 执行中发生 panic。',
                        action: '建议 provider 内捕获并返回 error，必要时开启 debug 栈。',
                    },
                    {
                        code: 'provider_timeout',
                        summary: 'Provider 执行超时。',
                        action: '优化初始化路径，或临时调大 WithProviderTimeout。',
                    },
                    {
                        code: 'inject_callback_error',
                        summary: 'Inject/TryInject 的回调函数主动返回了 error。',
                        action: '检查回调里的参数校验和业务逻辑分支。',
                    },
                    {
                        code: 'dependency_cycle',
                        summary: '检测到循环依赖。',
                        action: '拆分互相引用组件，改为接口或延迟注入解环。',
                    },
                    {
                        code: 'inject_failed',
                        summary: '注入失败兜底分类（需要结合 stage/message 细看）。',
                        action: '结合 root_cause + hint + provider_function 做链路定位。',
                    },
                ],
                diagnosticModalOpen: false,
                diagnosticModalType: 'runtime',
                diagnosticSearch: '',
                providerDetailModalOpen: false,
                providerModalNode: null,

                // Computed
                get filteredPackages() {
                    const query = this.packageSearch.toLowerCase();
                    return this.packages
                        .filter(p => p.name.toLowerCase().includes(query))
                        .sort((a, b) => b.provider_count - a.provider_count);
                },

                get totalProviders() {
                    return this.packages.reduce((sum, p) => sum + p.provider_count, 0);
                },

                get prefixSuggestions() {
                    return this.getPackagePrefixSuggestions();
                },

                get filteredRuntimeStats() {
                    const q = (this.runtimeStatsSearch || '').toLowerCase().trim();
                    return (this.runtimeStats || []).filter(s => {
                        if (this.runtimeStatsOnlyExecuted && Number(s.call_count || 0) <= 0) {
                            return false;
                        }
                        if (!q) {
                            return true;
                        }
                        const fn = String(s.function_name || '').toLowerCase();
                        const out = String(s.output_type || '').toLowerCase();
                        return fn.includes(q) || out.includes(q);
                    });
                },

                get filteredModalRuntimeStats() {
                    const q = (this.diagnosticSearch || '').toLowerCase().trim();
                    return (this.runtimeStats || []).filter(s => {
                        if (this.runtimeStatsOnlyExecuted && Number(s.call_count || 0) <= 0) {
                            return false;
                        }
                        if (!q) {
                            return true;
                        }
                        const fn = String(s.function_name || '').toLowerCase();
                        const out = String(s.output_type || '').toLowerCase();
                        const err = String(s.last_error || '').toLowerCase();
                        return fn.includes(q) || out.includes(q) || err.includes(q);
                    });
                },

                get filteredModalRecentErrors() {
                    const q = (this.diagnosticSearch || '').toLowerCase().trim();
                    return (this.recentErrors || []).filter(e => {
                        if (!q) {
                            return true;
                        }
                        const op = String(e.operation || '').toLowerCase();
                        const errType = String(e.error_type || '').toLowerCase();
                        const stage = String(e.stage || '').toLowerCase();
                        const comp = String(e.component || '').toLowerCase();
                        const providerFn = String(e.provider_function || '').toLowerCase();
                        const outType = String(e.output_type || '').toLowerCase();
                        const inputType = String(e.input_type || '').toLowerCase();
                        const root = String(e.root_cause || '').toLowerCase();
                        const hint = String(e.hint || '').toLowerCase();
                        const msg = String(e.message || '').toLowerCase();
                        return op.includes(q) || errType.includes(q) || stage.includes(q) || comp.includes(q) || providerFn.includes(q) || outType.includes(q) || inputType.includes(q) || root.includes(q) || hint.includes(q) || msg.includes(q);
                    });
                },

                get filteredModalErrorTypeGuide() {
                    const q = (this.diagnosticSearch || '').toLowerCase().trim();
                    return (this.errorTypeGuide || []).filter(item => {
                        if (!q) {
                            return true;
                        }
                        const code = String(item.code || '').toLowerCase();
                        const summary = String(item.summary || '').toLowerCase();
                        const action = String(item.action || '').toLowerCase();
                        return code.includes(q) || summary.includes(q) || action.includes(q);
                    });
                },

                get filteredModalTraceRecords() {
                    const q = (this.diagnosticSearch || '').toLowerCase().trim();
                    return (this.traceRecords || []).filter(item => {
                        if (!q) {
                            return true;
                        }
                        const traceId = String(item.trace_id || '').toLowerCase();
                        const spanId = String(item.span_id || '').toLowerCase();
                        const parent = String(item.parent_span_id || '').toLowerCase();
                        const op = String(item.operation || '').toLowerCase();
                        const ev = String(item.event || '').toLowerCase();
                        const status = String(item.status || '').toLowerCase();
                        const comp = String(item.component || '').toLowerCase();
                        const provider = String(item.provider_function || '').toLowerCase();
                        const output = String(item.output_type || '').toLowerCase();
                        const input = String(item.input_type || '').toLowerCase();
                        const err = String(item.error || '').toLowerCase();
                        return traceId.includes(q) || spanId.includes(q) || parent.includes(q) || op.includes(q) || ev.includes(q) || status.includes(q) || comp.includes(q) || provider.includes(q) || output.includes(q) || input.includes(q) || err.includes(q);
                    });
                },

                get traceGroups() {
                    const map = new Map();
                    (this.filteredModalTraceRecords || []).forEach(item => {
                        const traceId = item.trace_id || '';
                        if (!map.has(traceId)) {
                            map.set(traceId, {
                                traceId,
                                total: 0,
                                errorCount: 0,
                                records: [],
                            });
                        }
                        const g = map.get(traceId);
                        g.total++;
                        if (String(item.status || '').toLowerCase() === 'error') {
                            g.errorCount++;
                        }
                        g.records.push(item);
                    });

                    const groups = Array.from(map.values());
                    groups.forEach(g => {
                        g.records.sort((a, b) => Number(a.occurred_at_unix_nano || 0) - Number(b.occurred_at_unix_nano || 0));

                        const injectStart = g.records.find((rec) => String(rec.operation || '') === 'inject' && String(rec.event || '') === 'span.start');
                        g.injectFunction = '';
                        if (!g.injectFunction && injectStart) {
                            g.injectFunction = String(injectStart.component || '').trim();
                        }

                        // 构建 span 代表记录（优先 span.start）用于深度计算
                        const spanRep = new Map();
                        g.records.forEach((rec) => {
                            const sid = String(rec.span_id || '').trim();
                            if (!sid) return;

                            const prev = spanRep.get(sid);
                            if (!prev) {
                                spanRep.set(sid, rec);
                                return;
                            }

                            const curIsStart = String(rec.event || '') === 'span.start';
                            const prevIsStart = String(prev.event || '') === 'span.start';
                            if (curIsStart && !prevIsStart) {
                                spanRep.set(sid, rec);
                            }
                        });

                        const memo = new Map();
                        const computing = new Set();
                        const spanParent = new Map();
                        const spanOp = new Map();
                        const spanChildCount = new Map();
                        const spanEventCount = new Map();
                        const spanEnd = new Map();
                        const spanError = new Map();

                        spanRep.forEach((rec, sid) => {
                            spanOp.set(sid, String(rec.operation || '').trim());
                            const p = String(rec.parent_span_id || '').trim();
                            if (p) {
                                spanParent.set(sid, p);
                                spanChildCount.set(p, (spanChildCount.get(p) || 0) + 1);
                            }
                        });

                        const displayParentForSpan = (sid) => {
                            const parentSid = spanParent.get(sid) || '';
                            return parentSid;
                        };

                        g.records.forEach((rec) => {
                            const sid = String(rec.span_id || '').trim();
                            if (!sid) {
                                return;
                            }
                            spanEventCount.set(sid, (spanEventCount.get(sid) || 0) + 1);
                            if (String(rec.event || '') === 'span.end') {
                                spanEnd.set(sid, rec);
                            }
                            if (String(rec.status || '').toLowerCase() === 'error' || String(rec.error || '').trim()) {
                                if (!spanError.has(sid)) {
                                    spanError.set(sid, rec);
                                }
                            }
                        });

                        const depthForSpan = (sid) => {
                            sid = String(sid || '').trim();
                            if (!sid) return 0;
                            if (memo.has(sid)) return memo.get(sid);
                            if (computing.has(sid)) return 0;

                            const cur = spanRep.get(sid);
                            if (!cur) {
                                memo.set(sid, 0);
                                return 0;
                            }

                            computing.add(sid);
                            const parentSid = displayParentForSpan(sid);
                            let d = 0;
                            if (parentSid) {
                                d = depthForSpan(parentSid) + 1;
                            }
                            computing.delete(sid);
                            memo.set(sid, d);
                            return d;
                        };

                        g.treeRows = g.records.map((rec) => {
                            const sid = String(rec.span_id || '').trim();
                            const parentSid = String(rec.parent_span_id || '').trim();
                            let depth = 0;
                            if (sid) {
                                depth = depthForSpan(sid);
                            } else if (parentSid) {
                                depth = depthForSpan(parentSid) + 1;
                            }
                            return {
                                ...rec,
                                _depth: depth,
                                _canToggle: String(rec.event || '') === 'span.start' && !!sid && (spanChildCount.get(sid) || 0) > 0,
                            };
                        });

                        // 汇总 span 视图（一个 span 一行，默认更易读）
                        const allSpanRows = Array.from(spanRep.entries()).map(([sid, startRec]) => {
                            const endRec = spanEnd.get(sid);
                            const errRec = spanError.get(sid);
                            const status = (endRec && endRec.status) || (errRec ? 'error' : 'ok');
                            const parentSid = displayParentForSpan(sid);
                            const depth = depthForSpan(sid);

                            let durationNs = 0;
                            if (endRec && Number(endRec.duration_ns || 0) > 0) {
                                durationNs = Number(endRec.duration_ns || 0);
                            } else {
                                const st = Number(startRec.occurred_at_unix_nano || 0);
                                const et = endRec ? Number(endRec.occurred_at_unix_nano || 0) : 0;
                                if (st > 0 && et > st) {
                                    durationNs = et - st;
                                }
                            }

                            return {
                                trace_id: g.traceId,
                                span_id: sid,
                                parent_span_id: parentSid,
                                operation: startRec.operation || '',
                                component: startRec.component || '',
                                provider_function: startRec.provider_function || (startRec.attrs && (startRec.attrs.provider_function || startRec.attrs.provider_candidates || startRec.attrs.provider)) || '',
                                output_type: startRec.output_type || '',
                                input_type: startRec.input_type || '',
                                status: String(status || '').toLowerCase() || 'ok',
                                error: (errRec && (errRec.error || errRec.message)) || '',
                                duration_ns: durationNs,
                                occurred_at_unix_nano: Number(startRec.occurred_at_unix_nano || 0),
                                event_count: spanEventCount.get(sid) || 0,
                                _depth: depth,
                                _canToggle: (spanChildCount.get(sid) || 0) > 0,
                            };
                        });

                        g.spanErrorCount = allSpanRows.filter((row) => String(row.status || '').toLowerCase() === 'error' || String(row.error || '').trim()).length;
                        g.hasSpanError = g.spanErrorCount > 0;
                        let effectiveSpanRows = allSpanRows;
                        if (this.traceOnlyErrorSpan) {
                            effectiveSpanRows = effectiveSpanRows.filter((row) => row.status === 'error');
                        }

                        // 非树形：按时间排序
                        g.spanRows = [...effectiveSpanRows].sort((a, b) => Number(a.occurred_at_unix_nano || 0) - Number(b.occurred_at_unix_nano || 0));

                        // 树形：按父子关系前序遍历，保证展示顺序是“树状”而不是“时间流”
                        const rowBySpanID = new Map();
                        effectiveSpanRows.forEach((row) => {
                            const sid = String(row.span_id || '').trim();
                            if (!sid) return;
                            rowBySpanID.set(sid, row);
                        });

                        const childrenByParent = new Map();
                        const roots = [];
                        effectiveSpanRows.forEach((row) => {
                            const sid = String(row.span_id || '').trim();
                            if (!sid) return;

                            const parentSid = String(row.parent_span_id || '').trim();
                            if (parentSid && rowBySpanID.has(parentSid)) {
                                if (!childrenByParent.has(parentSid)) {
                                    childrenByParent.set(parentSid, []);
                                }
                                childrenByParent.get(parentSid).push(row);
                            } else {
                                roots.push(row);
                            }
                        });

                        const byOccurredAt = (a, b) => Number(a.occurred_at_unix_nano || 0) - Number(b.occurred_at_unix_nano || 0);
                        roots.sort(byOccurredAt);
                        for (const list of childrenByParent.values()) {
                            list.sort(byOccurredAt);
                        }

                        const treeOrdered = [];
                        const visited = new Set();
                        const walk = (node) => {
                            const sid = String((node && node.span_id) || '').trim();
                            if (!sid || visited.has(sid)) {
                                return;
                            }
                            visited.add(sid);
                            treeOrdered.push(node);
                            const children = childrenByParent.get(sid) || [];
                            children.forEach(walk);
                        };
                        roots.forEach(walk);
                        // 兜底：处理异常数据导致的孤立节点
                        effectiveSpanRows.forEach((row) => {
                            const sid = String(row.span_id || '').trim();
                            if (sid && !visited.has(sid)) {
                                walk(row);
                            }
                        });
                        g.spanRowsTreeOrdered = treeOrdered;

                        const isCollapsed = (traceId, sid) => !!this.traceCollapsedSpanKeys[this.traceSpanKey(traceId, sid)];
                        const hasCollapsedAncestor = (traceId, sid, includeSelf) => {
                            let cur = String(sid || '').trim();
                            if (!cur) return false;
                            if (!includeSelf) {
                                cur = displayParentForSpan(cur);
                            }
                            const guard = new Set();
                            while (cur && !guard.has(cur)) {
                                guard.add(cur);
                                if (isCollapsed(traceId, cur)) {
                                    return true;
                                }
                                cur = displayParentForSpan(cur);
                            }
                            return false;
                        };

                        g.visibleTreeRows = g.treeRows.filter((rec) => {
                            const sid = String(rec.span_id || '').trim();
                            const parentSid = String(rec.parent_span_id || '').trim();
                            if (sid) {
                                return !hasCollapsedAncestor(g.traceId, sid, false);
                            }
                            if (parentSid) {
                                return !hasCollapsedAncestor(g.traceId, parentSid, true);
                            }
                            return true;
                        });

                        g.visibleSpanRows = g.spanRowsTreeOrdered.filter((rec) => {
                            const sid = String(rec.span_id || '').trim();
                            const parentSid = String(rec.parent_span_id || '').trim();
                            if (sid) {
                                return !hasCollapsedAncestor(g.traceId, sid, false);
                            }
                            if (parentSid) {
                                return !hasCollapsedAncestor(g.traceId, parentSid, true);
                            }
                            return true;
                        });

                        const depthLimit = this.parseTraceDepthValue();
                        if (depthLimit > 0) {
                            g.treeRows = g.treeRows.filter((row) => Number(row._depth || 0) <= depthLimit);
                            g.visibleTreeRows = g.visibleTreeRows.filter((row) => Number(row._depth || 0) <= depthLimit);
                            g.spanRows = g.spanRows.filter((row) => Number(row._depth || 0) <= depthLimit);
                            g.spanRowsTreeOrdered = g.spanRowsTreeOrdered.filter((row) => Number(row._depth || 0) <= depthLimit);
                            g.visibleSpanRows = g.visibleSpanRows.filter((row) => Number(row._depth || 0) <= depthLimit);
                            g.records = g.treeRows;
                        }
                    });
                    // 按每条 trace 的 span 数量倒序；数量相同再按最近时间倒序
                    groups.sort((a, b) => {
                        const as = Array.isArray(a.spanRows) ? a.spanRows.length : 0;
                        const bs = Array.isArray(b.spanRows) ? b.spanRows.length : 0;
                        if (as !== bs) {
                            return bs - as;
                        }
                        const at = a.records.length ? Number(a.records[a.records.length - 1].occurred_at_unix_nano || 0) : 0;
                        const bt = b.records.length ? Number(b.records[b.records.length - 1].occurred_at_unix_nano || 0) : 0;
                        return bt - at;
                    });
                    return groups;
                },

                get traceSpanTotal() {
                    return (this.traceGroups || []).reduce((sum, g) => sum + (Array.isArray(g.spanRows) ? g.spanRows.length : 0), 0);
                },

                // Methods
                async init() {
                    this.loadLocalState();
                    await this.loadGroupRules();
                    await this.loadStats();
                    await this.loadPackages();
                    await this.loadDependencies();
                    await this.loadRuntimeStats();
                    await this.loadRecentErrors();
                    await this.loadTraceRecords();
                    if (window.mermaid && window.mermaid.initialize) {
                        window.mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });
                    }
                    document.addEventListener('fullscreenchange', () => this.handleFullscreenChange());
                },

                async loadGroupRules() {
                    try {
                        const res = await fetch(apiUrl('/api/group-rules'));
                        const rules = await res.json();
                        if (Array.isArray(rules) && rules.length > 0) {
                            if (!this.groupRules || this.groupRules.length === 0) {
                                this.groupRules = rules.map(g => ({
                                    name: g.name,
                                    prefixes: Array.isArray(g.prefixes) ? g.prefixes : [],
                                    _newPrefix: '',
                                    _rename: ''
                                }));
                                this.saveLocalState();
                            }
                        }
                    } catch (e) {
                        console.warn('加载分组清单失败:', e);
                    }
                },

                async loadStats() {
                    try {
                        const res = await fetch(apiUrl('/api/stats'));
                        this.stats = await res.json();
                    } catch (e) {
                        console.error('加载统计失败:', e);
                    }
                },

                async loadPackages() {
                    try {
                        const res = await fetch(apiUrl('/api/packages'));
                        this.packages = await res.json();
                    } catch (e) {
                        console.error('加载包列表失败:', e);
                    }
                },

                async loadRuntimeStats() {
                    this.runtimeStatsLoading = true;
                    this.runtimeStatsError = '';
                    try {
                        const res = await fetch(apiUrl('/api/runtime-stats'));
                        if (!res.ok) {
                            throw new Error('HTTP ' + res.status);
                        }
                        const data = await res.json();
                        this.runtimeStats = Array.isArray(data) ? data : [];
                        this.runtimeStatsUpdatedAt = new Date().toLocaleTimeString('zh-CN', { hour12: false });
                        if (this.network) {
                            this.rerenderCurrentGraph();
                        }
                    } catch (e) {
                        this.runtimeStatsError = '加载 provider 启动耗时失败';
                        console.error('加载 provider 启动耗时失败:', e);
                    } finally {
                        this.runtimeStatsLoading = false;
                    }
                },

                async loadRecentErrors() {
                    this.recentErrorsLoading = true;
                    this.recentErrorsError = '';
                    try {
                        const res = await fetch(apiUrl('/api/errors?limit=50'));
                        if (!res.ok) {
                            throw new Error('HTTP ' + res.status);
                        }
                        const data = await res.json();
                        this.recentErrors = Array.isArray(data) ? data : [];
                        this.recentErrorsUpdatedAt = new Date().toLocaleTimeString('zh-CN', { hour12: false });
                        if (this.network) {
                            this.rerenderCurrentGraph();
                        }
                    } catch (e) {
                        this.recentErrorsError = '加载最近错误失败';
                        console.error('加载最近错误失败:', e);
                    } finally {
                        this.recentErrorsLoading = false;
                    }
                },

                async loadTraceRecords() {
                    this.traceLoading = true;
                    this.traceError = '';
                    try {
                        const params = new URLSearchParams();
                        if (this.traceOperation) params.set('operation', this.traceOperation);
                        if (this.traceStatus) params.set('status', this.traceStatus);
                        if (this.traceEvent) params.set('event', this.traceEvent);
                        params.set('limit', String(this.traceLimit || 500));

                        const res = await fetch(apiUrl('/api/trace?' + params.toString()));
                        if (!res.ok) {
                            throw new Error('HTTP ' + res.status);
                        }
                        const data = await res.json();
                        this.traceRecords = Array.isArray(data.records) ? data.records : [];
                        this.traceUpdatedAt = new Date().toLocaleTimeString('zh-CN', { hour12: false });
                    } catch (e) {
                        this.traceError = '加载调用链 Trace 失败';
                        console.error('加载调用链 Trace 失败:', e);
                    } finally {
                        this.traceLoading = false;
                    }
                },

                openDiagnosticModal(type) {
                    if (type === 'errors') {
                        this.diagnosticModalType = 'errors';
                    } else if (type === 'trace') {
                        this.diagnosticModalType = 'trace';
                        this.diagnosticSearch = '';
                        this.loadTraceRecords();
                    } else if (type === 'error-guide') {
                        this.diagnosticModalType = 'error-guide';
                    } else {
                        this.diagnosticModalType = 'runtime';
                    }
                    if (type !== 'trace') {
                        this.diagnosticSearch = '';
                    }
                    this.diagnosticModalOpen = true;
                },

                closeDiagnosticModal() {
                    this.diagnosticModalOpen = false;
                    if (document.fullscreenElement) {
                        document.exitFullscreen().catch(() => { });
                    }
                    this.traceFullscreen = false;
                },

                handleFullscreenChange() {
                    const panel = this.$refs && this.$refs.diagnosticPanel;
                    this.traceFullscreen = !!(panel && document.fullscreenElement === panel);
                },

                async toggleTraceFullscreen() {
                    if (this.diagnosticModalType !== 'trace') {
                        return;
                    }
                    const panel = this.$refs && this.$refs.diagnosticPanel;
                    if (!panel) {
                        return;
                    }
                    try {
                        if (document.fullscreenElement === panel) {
                            await document.exitFullscreen();
                            this.traceFullscreen = false;
                        } else {
                            await panel.requestFullscreen();
                            this.traceFullscreen = true;
                        }
                    } catch (e) {
                        console.warn('切换 Trace 全屏失败:', e);
                    }
                },

                async focusSingleTrace(traceId) {
                    const id = String(traceId || '').trim();
                    if (!id || this.diagnosticModalType !== 'trace') {
                        return;
                    }

                    this.diagnosticSearch = id;
                    this.traceCollapsedGroupKeys = {};
                    this.traceCollapsedSpanKeys = {};

                    if (!this.traceFullscreen) {
                        await this.toggleTraceFullscreen();
                    }
                },

                refreshTraceFromModal() {
                    this.loadTraceRecords();
                },

                isSingleTraceView() {
                    return this.diagnosticModalType === 'trace' && Array.isArray(this.traceGroups) && this.traceGroups.length === 1;
                },

                parseTraceDepthValue() {
                    const raw = String(this.traceDepth ?? '').trim();
                    let depth = Number.parseInt(raw, 10);
                    if (!Number.isFinite(depth) || depth < 0) {
                        depth = 0;
                    }
                    if (depth > 256) {
                        depth = 256;
                    }
                    return depth;
                },

                traceProviderLabel(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }

                    const direct = String(item.provider_function || '').trim();
                    if (direct) {
                        return direct;
                    }

                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const providerList = Array.isArray(attrs.provider_functions)
                        ? attrs.provider_functions.map((v) => String(v || '').trim()).filter((v) => v.length > 0)
                        : [];
                    if (providerList.length > 1) {
                        return `${providerList[0]} (+${providerList.length - 1} more)`;
                    }
                    if (providerList.length === 1) {
                        return providerList[0];
                    }

                    const fromAttrs = [attrs.provider_function, attrs.provider_candidates, attrs.provider]
                        .map((v) => String(v || '').trim())
                        .find((v) => v.length > 0);
                    if (fromAttrs) {
                        return fromAttrs;
                    }

                    return String(item.component || '').trim();
                },

                traceProviderFunctions(item) {
                    if (!item || typeof item !== 'object') {
                        return [];
                    }

                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const list = Array.isArray(attrs.provider_functions)
                        ? attrs.provider_functions.map((v) => String(v || '').trim()).filter((v) => v.length > 0)
                        : [];
                    if (list.length > 0) {
                        return list;
                    }

                    const fromCandidates = String(attrs.provider_candidates || '').trim();
                    if (fromCandidates) {
                        return fromCandidates.split(',').map((v) => v.trim()).filter((v) => v.length > 0);
                    }

                    const single = this.traceProviderLabel(item);
                    return single ? [single] : [];
                },

                traceInputType(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }
                    if (String(item.input_type || '').trim()) {
                        return String(item.input_type || '').trim();
                    }
                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const declared = String(attrs.input_type || '').trim();
                    const resolved = String(attrs.resolved_input_type || '').trim();
                    if (declared) {
                        return resolved && resolved !== declared ? `${declared} (resolved: ${resolved})` : declared;
                    }
                    return '';
                },

                traceOutputType(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }
                    if (String(item.output_type || '').trim()) {
                        return String(item.output_type || '').trim();
                    }
                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    return String(attrs.output_type || '').trim();
                },

                traceTypeMeta(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }
                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const parts = [];
                    const queryKind = String(attrs.query_kind || '').trim();
                    if (queryKind) {
                        parts.push(`query_kind: ${queryKind}`);
                    }
                    if (attrs.index !== undefined && attrs.index !== null && String(attrs.index) !== '') {
                        parts.push(`index: ${attrs.index}`);
                    }
                    if (attrs.aggregate_input === true) {
                        parts.push('aggregate_input: true');
                    }
                    return parts.join(' · ');
                },

                firstNonEmpty(...values) {
                    for (const v of values) {
                        const s = String(v ?? '').trim();
                        if (s) {
                            return s;
                        }
                    }
                    return '';
                },

                traceErrorText(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }
                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    return this.firstNonEmpty(
                        item.error,
                        item.message,
                        attrs.error,
                        attrs.message,
                        attrs.root_cause,
                        attrs.cause,
                    );
                },

                traceErrorContext(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }
                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const parts = [];

                    const stage = this.firstNonEmpty(attrs.stage);
                    if (stage) parts.push(`stage=${stage}`);

                    const queryKind = this.firstNonEmpty(attrs.query_kind);
                    if (queryKind) parts.push(`query=${queryKind}`);

                    if (attrs.index !== undefined && attrs.index !== null && String(attrs.index).trim() !== '') {
                        parts.push(`index=${attrs.index}`);
                    }

                    const resultPath = this.firstNonEmpty(attrs.result_path);
                    if (resultPath) parts.push(`path=${resultPath}`);

                    const timedOut = attrs.timed_out;
                    if (timedOut === true || String(timedOut).toLowerCase() === 'true') {
                        parts.push('timed_out=true');
                    }

                    const hint = this.firstNonEmpty(attrs.hint);
                    if (hint) parts.push(`hint=${hint}`);

                    const root = this.firstNonEmpty(attrs.root_cause);
                    if (root) parts.push(`root=${root}`);

                    return parts.join(' · ');
                },

                toMultilineText(text) {
                    const s = String(text || '').trim();
                    if (!s) {
                        return '';
                    }
                    return s
                        .replace(/\s*\|\s*/g, '\n')
                        .replace(/\s*·\s*/g, '\n')
                        .replace(/\s*;\s*/g, '\n')
                        .replace(/\s+,\s+(?=(stage|query|index|path|hint|root|timed_out)=)/gi, '\n')
                        .replace(/\s+caused by:\s+/gi, '\ncaused by: ')
                        .trim();
                },

                traceErrorMultilineText(item) {
                    return this.toMultilineText(this.traceErrorText(item));
                },

                traceErrorContextMultiline(item) {
                    return this.toMultilineText(this.traceErrorContext(item));
                },

                extractPkgPathFromSymbol(symbol) {
                    const s = String(symbol || '').trim();
                    if (!s) {
                        return '';
                    }

                    const lastDot = s.lastIndexOf('.');
                    if (lastDot > 0) {
                        const pkg = s.slice(0, lastDot).trim();
                        if (pkg.includes('/')) {
                            return pkg;
                        }
                        if (pkg.includes('.') && !pkg.includes('(')) {
                            return pkg;
                        }
                    }

                    if (s.includes('/')) {
                        return s;
                    }

                    return '';
                },

                tracePackagePathLabel(item) {
                    if (!item || typeof item !== 'object') {
                        return '';
                    }

                    const attrs = item.attrs && typeof item.attrs === 'object' ? item.attrs : {};
                    const candidates = [];

                    candidates.push(item.component);
                    candidates.push(item.provider_function);
                    candidates.push(this.traceProviderLabel(item));
                    candidates.push(this.traceInputType(item));
                    candidates.push(this.traceOutputType(item));
                    candidates.push(attrs.provider);
                    candidates.push(attrs.provider_function);
                    candidates.push(attrs.provider_candidates);

                    const providerFns = this.traceProviderFunctions(item);
                    for (const fn of providerFns) {
                        candidates.push(fn);
                    }

                    const seen = new Set();
                    const pkgList = [];
                    for (const c of candidates) {
                        const pkg = this.extractPkgPathFromSymbol(c);
                        if (!pkg || seen.has(pkg)) {
                            continue;
                        }
                        seen.add(pkg);
                        pkgList.push(pkg);
                    }

                    return pkgList.join(' | ');
                },

                traceIndentStyle(item) {
                    const depth = Math.max(0, Number((item && item._depth) || 0));
                    const mode = String(this.traceIndentMode || 'log');

                    // 线性模式：保留传统层级感
                    if (mode === 'linear') {
                        const px = Math.min(depth, 64) * 16;
                        return `margin-left:${px}px`;
                    }

                    // 对数模式：深链路不会横向爆炸，适合大型项目
                    const px = Math.round(Math.log2(depth + 1) * 42);
                    return `margin-left:${px}px`;
                },

                traceSpanKey(traceId, spanId) {
                    return `${String(traceId || '')}::${String(spanId || '')}`;
                },

                traceGroupKey(traceId) {
                    return String(traceId || '');
                },

                isTraceGroupCollapsed(traceId) {
                    return !!this.traceCollapsedGroupKeys[this.traceGroupKey(traceId)];
                },

                toggleTraceGroupCollapse(traceId) {
                    const key = this.traceGroupKey(traceId);
                    const next = { ...(this.traceCollapsedGroupKeys || {}) };
                    if (next[key]) {
                        delete next[key];
                    } else {
                        next[key] = true;
                    }
                    this.traceCollapsedGroupKeys = next;
                },

                collapseAllTraceGroups() {
                    const next = {};
                    (this.traceGroups || []).forEach((g) => {
                        next[this.traceGroupKey(g.traceId)] = true;
                    });
                    this.traceCollapsedGroupKeys = next;
                },

                expandAllTraceGroups() {
                    this.traceCollapsedGroupKeys = {};
                },

                isTraceSpanCollapsed(traceId, spanId) {
                    if (!traceId || !spanId) {
                        return false;
                    }
                    return !!this.traceCollapsedSpanKeys[this.traceSpanKey(traceId, spanId)];
                },

                toggleTraceSpanCollapse(traceId, spanId) {
                    if (!traceId || !spanId) {
                        return;
                    }
                    const key = this.traceSpanKey(traceId, spanId);
                    const next = { ...(this.traceCollapsedSpanKeys || {}) };
                    if (next[key]) {
                        delete next[key];
                    } else {
                        next[key] = true;
                    }
                    this.traceCollapsedSpanKeys = next;
                },

                openProviderDetailModal(node) {
                    if (!node || !node.data || node.data.type !== 'provider') {
                        return;
                    }
                    this.providerModalNode = node;
                    this.providerDetailModalOpen = true;
                },

                providerModalError() {
                    if (!this.providerModalNode || !this.providerModalNode.data) {
                        return null;
                    }
                    return this.getProviderErrorFor(this.providerModalNode.data);
                },

                providerPrettyJson(node) {
                    try {
                        return JSON.stringify(node || {}, null, 2);
                    } catch (e) {
                        return String(node || '');
                    }
                },

                async selectPackage(pkgName) {
                    this.currentPackage = pkgName;
                    await this.loadDependencies();
                },

                async loadDependencies() {
                    this.loading = true;
                    try {
                        let url = apiUrl('/api/dependencies');
                        if (this.currentPackage) {
                            url += '?package=' + encodeURIComponent(this.currentPackage);
                        }
                        const res = await fetch(url);
                        this.allData = await res.json();
                        this.renderGraph();
                    } catch (e) {
                        console.error('加载依赖失败:', e);
                    } finally {
                        this.loading = false;
                    }
                },

                switchView(view) {
                    this.currentView = view;
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.renderGraph();
                },

                resetView() {
                    this.currentPackage = null;
                    this.currentView = 'providers';
                    this.packageSearch = '';
                    this.globalSearch = '';
                    this.searchResults = [];
                    this.selectedNode = null;
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.saveLocalState();
                    this.loadDependencies();
                },

                onDepthChange() {
                    this.normalizeCurrentDepth(2);
                    if (this.focusedType) {
                        if (this.focusedGraph === 'dependency') {
                            this.showDependencyGraph(this.focusedType, 'type');
                            return;
                        }
                        if (this.focusedGraph === 'type') {
                            this.focusOnType(this.focusedType);
                            return;
                        }
                    }
                    if (this.focusedGraph === 'group' && this.focusedGroup) {
                        this.showGroupGraph(this.focusedGroup);
                        return;
                    }
                    this.renderGraph();
                },

                parseDepthValue(defaultDepth = 2) {
                    const raw = String(this.currentDepth ?? '').trim();
                    let depth = Number.parseInt(raw, 10);
                    if (!Number.isFinite(depth) || depth < 0) {
                        depth = defaultDepth;
                    }
                    // 防止极端值导致页面卡顿
                    if (depth > 128) {
                        depth = 128;
                    }
                    return depth;
                },

                normalizeCurrentDepth(defaultDepth = 2) {
                    const depth = this.parseDepthValue(defaultDepth);
                    this.currentDepth = String(depth);
                    return depth;
                },

                rerenderCurrentGraph() {
                    if (this.focusedType) {
                        if (this.focusedGraph === 'dependency') {
                            this.showDependencyGraph(this.focusedType, 'type');
                            return;
                        }
                        if (this.focusedGraph === 'type') {
                            this.focusOnType(this.focusedType);
                            return;
                        }
                    }
                    if (this.focusedGraph === 'group' && this.focusedGroup) {
                        this.showGroupGraph(this.focusedGroup);
                        return;
                    }
                    this.renderGraph();
                },

                isGroupExpanded(groupName) {
                    return this.expandedGroups.includes(groupName);
                },

                toggleGroupExpand(groupName) {
                    if (!groupName) return;
                    if (this.isGroupExpanded(groupName)) {
                        this.expandedGroups = this.expandedGroups.filter(g => g !== groupName);
                    } else {
                        this.expandedGroups = [...this.expandedGroups, groupName];
                    }
                    this.rerenderCurrentGraph();
                },

                // Global search functionality
                onGlobalSearchInput() {
                    if (!this.allData || this.globalSearch.length < 1) {
                        this.searchResults = [];
                        return;
                    }

                    const query = this.globalSearch.toLowerCase();
                    const results = [];
                    const seen = new Set();

                    // Search in providers
                    this.allData.providers.forEach(provider => {
                        const fnName = provider.function_name || '';
                        const outputTypes = this.providerOutputTypes(provider);
                        const outputType = outputTypes[0] || '';
                        const outputsText = outputTypes.join(' | ');

                        if (fnName.toLowerCase().includes(query) || outputsText.toLowerCase().includes(query)) {
                            if (!seen.has(provider.id)) {
                                seen.add(provider.id);
                                results.push({
                                    id: provider.id,
                                    type: 'provider',
                                    label: this.formatFunctionName(fnName),
                                    fullName: fnName,
                                    outputType: outputType,
                                    data: provider
                                });
                            }
                        }

                        // Also search input types
                        (provider.input_types || []).forEach(inputType => {
                            if (inputType.toLowerCase().includes(query) && !seen.has(inputType)) {
                                seen.add(inputType);
                                results.push({
                                    id: inputType,
                                    type: 'type',
                                    label: this.formatTypeName(inputType),
                                    fullName: inputType,
                                    data: { type: 'type', fullType: inputType }
                                });
                            }
                        });

                        // Search output types
                        outputTypes.forEach(outType => {
                            if (!outType || !outType.toLowerCase().includes(query) || seen.has(outType)) {
                                return;
                            }
                            seen.add(outType);
                            results.push({
                                id: outType,
                                type: 'type',
                                label: this.formatTypeName(outType),
                                fullName: outType,
                                data: { type: 'type', fullType: outType }
                            });
                        });
                    });

                    // Limit results
                    this.searchResults = results.slice(0, 20);
                },

                selectSearchResult(result) {
                    this.globalSearch = '';
                    this.searchResults = [];

                    if (result.type === 'type') {
                        // 类型：显示依赖关系图
                        this.showDependencyGraph(result.fullName, 'type');
                    } else if (result.type === 'provider') {
                        // Provider：以其输出类型为中心显示依赖图
                        const outputType = this.primaryOutputType(result.data || {}) || result.outputType;
                        if (outputType) {
                            this.showDependencyGraph(outputType, 'type');
                        }
                        this.selectedNode = { data: result.data };
                    }
                },

                selectFirstSearchResult() {
                    if (this.searchResults.length > 0) {
                        this.selectSearchResult(this.searchResults[0]);
                    }
                },

                // 显示以某个类型为中心的依赖图（包含依赖者和被依赖者）
                showDependencyGraph(targetType, nodeType) {
                    if (!this.allData) return;

                    this.focusedType = targetType;
                    this.focusedGraph = 'dependency';
                    this.focusedGroup = null;

                    const depth = this.parseDepthValue(2);
                    const nodes = [];
                    const edges = [];
                    const nodeMap = new Map();
                    const typePkgMap = this.buildTypePkgMap();

                    // 构建类型到 Provider 的映射
                    const typeToProviders = new Map(); // outputType -> providers
                    const typeToConsumers = new Map(); // inputType -> providers that use it

                    this.allData.providers.forEach(provider => {
                        const outputTypes = this.providerOutputTypes(provider);
                        outputTypes.forEach(outputType => {
                            if (!typeToProviders.has(outputType)) {
                                typeToProviders.set(outputType, []);
                            }
                            typeToProviders.get(outputType).push(provider);
                        });

                        (provider.input_types || []).forEach(inputType => {
                            if (!typeToConsumers.has(inputType)) {
                                typeToConsumers.set(inputType, []);
                            }
                            typeToConsumers.get(inputType).push(provider);
                        });
                    });

                    // BFS 向上查找依赖（当前类型依赖什么）
                    const findDependencies = (startType, maxDepth) => {
                        const visited = new Set();
                        const queue = [{ type: startType, level: 0 }];
                        const result = [];

                        while (queue.length > 0) {
                            const { type, level } = queue.shift();
                            if (visited.has(type) || level > maxDepth) continue;
                            visited.add(type);

                            result.push({ type, level, direction: 'dependency' });

                            // 找到生产这个类型的 providers，获取它们的输入类型
                            const providers = typeToProviders.get(type) || [];
                            providers.forEach(p => {
                                (p.input_types || []).forEach(inputType => {
                                    if (!visited.has(inputType)) {
                                        queue.push({ type: inputType, level: level + 1 });
                                    }
                                });
                            });
                        }
                        return result;
                    };

                    // BFS 向下查找被依赖者（什么依赖当前类型）
                    const findDependents = (startType, maxDepth) => {
                        const visited = new Set();
                        const queue = [{ type: startType, level: 0 }];
                        const result = [];

                        while (queue.length > 0) {
                            const { type, level } = queue.shift();
                            if (visited.has(type) || level > maxDepth) continue;
                            visited.add(type);

                            if (level > 0) { // 不重复添加起始节点
                                result.push({ type, level, direction: 'dependent' });
                            }

                            // 找到使用这个类型作为输入的 providers，获取它们的输出类型
                            const consumers = typeToConsumers.get(type) || [];
                            consumers.forEach(p => {
                                this.providerOutputTypes(p).forEach(outputType => {
                                    if (outputType && !visited.has(outputType)) {
                                        queue.push({ type: outputType, level: level + 1 });
                                    }
                                });
                            });
                        }
                        return result;
                    };

                    // 获取依赖和被依赖者
                    const dependencies = findDependencies(targetType, depth);
                    const dependents = findDependents(targetType, depth);

                    // 添加所有节点
                    const allTypes = [...dependencies, ...dependents];
                    allTypes.forEach(({ type, level, direction }) => {
                        if (!nodeMap.has(type)) {
                            const isTarget = type === targetType;
                            let bgColor, borderColor;

                            if (isTarget) {
                                bgColor = '#fde68a'; borderColor = '#f59e0b'; // 目标：黄色
                            } else if (direction === 'dependency') {
                                bgColor = '#bbf7d0'; borderColor = '#22c55e'; // 依赖：绿色
                            } else {
                                bgColor = '#fecaca'; borderColor = '#ef4444'; // 被依赖：红色
                            }

                            nodes.push({
                                id: type,
                                label: this.formatTypeName(type),
                                title: `类型: ${type}\n层级: ${level}\n${direction === 'dependency' ? '← 依赖' : '→ 被依赖'}`,
                                color: { background: bgColor, border: borderColor },
                                level: direction === 'dependency' ? -level : level,
                                data: { type: 'type', fullType: type, packagePath: typePkgMap.get(type) || '' }
                            });
                            nodeMap.set(type, true);
                        }
                    });

                    // 添加边
                    this.allData.providers.forEach(provider => {
                        const outputType = provider.output_type;
                        (provider.input_types || []).forEach(inputType => {
                            if (nodeMap.has(inputType) && nodeMap.has(outputType)) {
                                const edgeId = `${inputType}->${outputType}`;
                                if (!nodeMap.has(edgeId)) {
                                    nodeMap.set(edgeId, true);
                                    edges.push({
                                        from: inputType,
                                        to: outputType,
                                        arrows: 'to',
                                        color: { color: '#9ca3af' }
                                    });
                                }
                            }
                        });
                    });

                    // 渲染图形
                    const aggregated = this.aggregateByGroups(nodes, edges, new Set([targetType]));
                    const filteredByPrefix = this.filterByPrefix(aggregated.nodes, aggregated.edges, new Set([targetType]));

                    const container = document.getElementById('network');
                    const graphData = {
                        nodes: new vis.DataSet(filteredByPrefix.nodes),
                        edges: new vis.DataSet(filteredByPrefix.edges)
                    };
                    this.lastGraphData = graphData;
                    this.lastGraphData = graphData;
                    this.lastGraphData = graphData;

                    if (this.network) {
                        this.network.destroy();
                    }

                    this.network = new vis.Network(container, graphData, this.getNetworkOptions());

                    // 聚焦到目标节点
                    setTimeout(() => {
                        try {
                            this.network.selectNodes([targetType]);
                            this.network.focus(targetType, { scale: 1, animation: true });
                        } catch (e) { }
                    }, 100);

                    this.network.on('click', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = graphData.nodes.get(nodeId);
                            this.selectedNode = node;
                            if (node && node.data && node.data.type === 'provider') {
                                this.openProviderDetailModal(node);
                            }
                        }
                    });

                    // 双击展开该节点的依赖
                    this.network.on('doubleClick', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = graphData.nodes.get(nodeId);
                            if (node && node.data && node.data.type === 'group') {
                                this.toggleGroupExpand(node.data.group);
                                return;
                            }
                            if (node && node.data && node.data.fullType) {
                                this.showDependencyGraph(node.data.fullType, 'type');
                            }
                        }
                    });
                },

                renderGraph() {
                    if (!this.allData) return;

                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.focusedGroup = null;

                    const nodes = [];
                    const edges = [];
                    const nodeMap = new Map();
                    const typePkgMap = this.buildTypePkgMap();

                    if (this.currentView === 'providers') {
                        this.allData.providers.forEach((provider) => {
                            const providerNodeId = provider.id;
                            const fnName = this.providerNodeLabel(provider);
                            const outputTypes = this.providerOutputTypes(provider);

                            // Provider node
                            if (!nodeMap.has(providerNodeId)) {
                                nodes.push({
                                    id: providerNodeId,
                                    label: fnName,
                                    title: this.buildProviderTooltip(provider),
                                    color: this.getProviderNodeColor(provider),
                                    shape: 'box',
                                    font: { size: 11 },
                                    data: { ...provider, type: 'provider', packagePath: provider.function_pkg || '' }
                                });
                                nodeMap.set(providerNodeId, true);
                            }

                            // Output type nodes + Provider -> Output edges
                            outputTypes.forEach((outType) => {
                                if (!nodeMap.has(outType)) {
                                    nodes.push({
                                        id: outType,
                                        label: this.formatTypeName(outType),
                                        title: '类型: ' + outType,
                                        color: { background: '#bfdbfe', border: '#3b82f6' },
                                        shape: 'ellipse',
                                        font: { size: 10 },
                                        data: { type: 'type', fullType: outType, packagePath: typePkgMap.get(outType) || provider.output_pkg || '' }
                                    });
                                    nodeMap.set(outType, true);
                                }

                                edges.push({
                                    from: providerNodeId,
                                    to: outType,
                                    arrows: 'to',
                                    color: { color: '#22c55e' }
                                });
                            });

                            // Input types
                            (provider.input_types || []).forEach(inputType => {
                                if (!nodeMap.has(inputType)) {
                                    nodes.push({
                                        id: inputType,
                                        label: this.formatTypeName(inputType),
                                        title: '类型: ' + inputType,
                                        color: { background: '#bfdbfe', border: '#3b82f6' },
                                        shape: 'ellipse',
                                        font: { size: 10 },
                                        data: { type: 'type', fullType: inputType, packagePath: typePkgMap.get(inputType) || '' }
                                    });
                                    nodeMap.set(inputType, true);
                                }

                                edges.push({
                                    from: inputType,
                                    to: providerNodeId,
                                    arrows: 'to',
                                    dashes: true,
                                    color: { color: '#f59e0b' }
                                });
                            });
                        });
                    } else {
                        // Types view
                        this.allData.edges.forEach(edge => {
                            if (edge.type !== 'provider') return;

                            if (!nodeMap.has(edge.from)) {
                                nodes.push({
                                    id: edge.from,
                                    label: this.formatTypeName(edge.from),
                                    title: '类型: ' + edge.from,
                                    color: { background: '#bfdbfe', border: '#3b82f6' },
                                    data: { type: 'type', fullType: edge.from, packagePath: typePkgMap.get(edge.from) || '' }
                                });
                                nodeMap.set(edge.from, true);
                            }

                            if (!nodeMap.has(edge.to)) {
                                nodes.push({
                                    id: edge.to,
                                    label: this.formatTypeName(edge.to),
                                    title: '类型: ' + edge.to,
                                    color: { background: '#bfdbfe', border: '#3b82f6' },
                                    data: { type: 'type', fullType: edge.to, packagePath: typePkgMap.get(edge.to) || '' }
                                });
                                nodeMap.set(edge.to, true);
                            }

                            edges.push({
                                from: edge.from,
                                to: edge.to,
                                arrows: 'to',
                                color: { color: '#9ca3af' }
                            });
                        });
                    }

                    // Create network
                    const aggregated = this.aggregateByGroups(nodes, edges);
                    const filteredByPrefix = this.filterByPrefix(aggregated.nodes, aggregated.edges);

                    const container = document.getElementById('network');
                    const data = {
                        nodes: new vis.DataSet(filteredByPrefix.nodes),
                        edges: new vis.DataSet(filteredByPrefix.edges)
                    };
                    this.lastGraphData = data;

                    const options = this.getNetworkOptions();

                    if (this.network) {
                        this.network.destroy();
                    }

                    this.network = new vis.Network(container, data, options);

                    // Click event
                    this.network.on('click', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = data.nodes.get(nodeId);
                            this.selectedNode = node;
                            if (node && node.data && node.data.type === 'provider') {
                                this.openProviderDetailModal(node);
                            }
                        }
                    });

                    // Double click to focus
                    this.network.on('doubleClick', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = data.nodes.get(nodeId);
                            if (node && node.data && node.data.type === 'group') {
                                this.toggleGroupExpand(node.data.group);
                                return;
                            }
                            if (node && node.data && node.data.type === 'type') {
                                this.focusOnType(node.data.fullType);
                            }
                        }
                    });
                },

                getNetworkOptions(forceHierarchical = false) {
                    const isHierarchical = forceHierarchical || this.currentLayout === 'hierarchical';
                    const levelSeparation = 120;
                    const nodeSpacing = 150;
                    const treeSpacing = 150;
                    return {
                        nodes: {
                            shape: 'box',
                            font: { size: 12, face: 'system-ui, sans-serif' },
                            borderWidth: 2,
                            shadow: { enabled: true, size: 5, x: 2, y: 2 },
                            margin: 8
                        },
                        edges: {
                            width: 1.5,
                            smooth: isHierarchical
                                ? { type: 'cubicBezier', roundness: 0.4 }
                                : { type: 'continuous' }
                        },
                        physics: {
                            enabled: !isHierarchical,
                            stabilization: { iterations: 150 },
                            barnesHut: {
                                gravitationalConstant: -2000,
                                springLength: 150
                            }
                        },
                        interaction: {
                            hover: true,
                            tooltipDelay: 100,
                            zoomView: true,
                            dragView: true
                        },
                        layout: isHierarchical ? {
                            hierarchical: {
                                enabled: true,
                                direction: 'UD',
                                sortMethod: 'directed',
                                levelSeparation,
                                nodeSpacing,
                                treeSpacing
                            }
                        } : {
                            hierarchical: { enabled: false }
                        }
                    };
                },

                async focusOnType(typeName) {
                    this.loading = true;
                    this.focusedType = typeName;
                    this.focusedGraph = 'type';
                    this.focusedGroup = null;
                    try {
                        const depth = this.parseDepthValue(0);
                        const url = apiUrl(`/api/type/${encodeURIComponent(typeName)}?depth=${depth}`);
                        const res = await fetch(url);
                        const data = await res.json();
                        this.renderTypeGraph(data);
                    } catch (e) {
                        console.error('加载类型详情失败:', e);
                    } finally {
                        this.loading = false;
                    }
                },

                renderTypeGraph(data) {
                    const nodes = [];
                    const edges = [];
                    const nodeMap = new Map();
                    const typePkgMap = this.buildTypePkgMap();

                    data.nodes.forEach(node => {
                        if (!nodeMap.has(node.id)) {
                            const isRoot = node.id === data.root_type;
                            nodes.push({
                                id: node.id,
                                label: this.formatTypeName(node.type),
                                title: `类型: ${node.type}\n包: ${node.package}\n层级: ${node.level}`,
                                color: isRoot
                                    ? { background: '#fde68a', border: '#f59e0b' }
                                    : { background: '#bfdbfe', border: '#3b82f6' },
                                level: node.level,
                                data: { type: 'type', fullType: node.type, packagePath: typePkgMap.get(node.type) || node.package || '' }
                            });
                            nodeMap.set(node.id, true);
                        }
                    });

                    data.edges.forEach(edge => {
                        edges.push({
                            from: edge.from,
                            to: edge.to,
                            arrows: 'to',
                            color: { color: '#9ca3af' }
                        });
                    });

                    const aggregated = this.aggregateByGroups(nodes, edges, new Set([data.root_type]));
                    const filteredByPrefix = this.filterByPrefix(aggregated.nodes, aggregated.edges, new Set([data.root_type]));

                    const container = document.getElementById('network');
                    const graphData = {
                        nodes: new vis.DataSet(filteredByPrefix.nodes),
                        edges: new vis.DataSet(filteredByPrefix.edges)
                    };

                    if (this.network) {
                        this.network.destroy();
                    }

                    this.network = new vis.Network(container, graphData, this.getNetworkOptions());

                    this.network.on('click', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = graphData.nodes.get(nodeId);
                            this.selectedNode = node;
                            if (node && node.data && node.data.type === 'provider') {
                                this.openProviderDetailModal(node);
                            }
                        }
                    });

                    this.network.on('doubleClick', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = graphData.nodes.get(nodeId);
                            if (node && node.data && node.data.type === 'group') {
                                this.toggleGroupExpand(node.data.group);
                            }
                        }
                    });
                },

                showGroupGraph(groupName) {
                    if (!groupName || !this.groupMembers || !this.groupMembers[groupName]) return;
                    this.focusedGraph = 'group';
                    this.focusedGroup = groupName;
                    const memberList = this.groupMembers[groupName] || [];
                    if (memberList.length === 0) return;

                    const depth = this.parseDepthValue(0);
                    const depthLimit = depth > 0 ? depth : Number.POSITIVE_INFINITY;

                    const memberIds = new Set(memberList.map(m => m.id));
                    const nodes = [];
                    const edges = [];
                    const nodeMap = new Map();
                    const typePkgMap = this.buildTypePkgMap();

                    // Build full graph nodes and edges
                    (this.allData.providers || []).forEach(provider => {
                        const providerNodeId = provider.id;
                        if (!nodeMap.has(providerNodeId)) {
                            nodes.push({
                                id: providerNodeId,
                                label: this.providerNodeLabel(provider),
                                title: this.buildProviderTooltip(provider),
                                color: this.getProviderNodeColor(provider),
                                shape: 'box',
                                font: { size: 11 },
                                data: { ...provider, type: 'provider', packagePath: provider.function_pkg || '' }
                            });
                            nodeMap.set(providerNodeId, true);
                        }

                        this.providerOutputTypes(provider).forEach((outType) => {
                            if (!nodeMap.has(outType)) {
                                nodes.push({
                                    id: outType,
                                    label: this.formatTypeName(outType),
                                    title: '类型: ' + outType,
                                    color: { background: '#bfdbfe', border: '#3b82f6' },
                                    shape: 'ellipse',
                                    font: { size: 10 },
                                    data: { type: 'type', fullType: outType, packagePath: typePkgMap.get(outType) || provider.output_pkg || '' }
                                });
                                nodeMap.set(outType, true);
                            }

                            edges.push({
                                from: providerNodeId,
                                to: outType,
                                arrows: 'to',
                                color: { color: '#22c55e' }
                            });
                        });

                        (provider.input_types || []).forEach(inputType => {
                            if (!nodeMap.has(inputType)) {
                                nodes.push({
                                    id: inputType,
                                    label: this.formatTypeName(inputType),
                                    title: '类型: ' + inputType,
                                    color: { background: '#bfdbfe', border: '#3b82f6' },
                                    shape: 'ellipse',
                                    font: { size: 10 },
                                    data: { type: 'type', fullType: inputType, packagePath: typePkgMap.get(inputType) || '' }
                                });
                                nodeMap.set(inputType, true);
                            }

                            edges.push({
                                from: inputType,
                                to: providerNodeId,
                                arrows: 'to',
                                dashes: true,
                                color: { color: '#f59e0b' }
                            });
                        });
                    });

                    // BFS from group members to include upstream/downstream within depth
                    const adjacency = new Map();
                    nodes.forEach(n => adjacency.set(n.id, []));
                    edges.forEach(e => {
                        if (adjacency.has(e.from)) adjacency.get(e.from).push(e.to);
                        if (adjacency.has(e.to)) adjacency.get(e.to).push(e.from);
                    });

                    const keep = new Set();
                    const queue = Array.from(memberIds).map(id => ({ id, level: 0 }));
                    while (queue.length > 0) {
                        const { id, level } = queue.shift();
                        if (keep.has(id) || level > depthLimit) continue;
                        keep.add(id);
                        const neighbors = adjacency.get(id) || [];
                        neighbors.forEach(next => {
                            if (!keep.has(next)) {
                                queue.push({ id: next, level: level + 1 });
                            }
                        });
                    }

                    const filteredNodes = nodes.filter(n => keep.has(n.id));
                    const filteredEdges = edges.filter(e => keep.has(e.from) && keep.has(e.to));

                    const container = document.getElementById('network');
                    const filteredByPrefix = this.filterByPrefix(filteredNodes, filteredEdges, memberIds);
                    const graphData = {
                        nodes: new vis.DataSet(filteredByPrefix.nodes),
                        edges: new vis.DataSet(filteredByPrefix.edges)
                    };

                    if (this.network) {
                        this.network.destroy();
                    }

                    this.network = new vis.Network(container, graphData, this.getNetworkOptions(true));
                    this.network.on('click', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = graphData.nodes.get(nodeId);
                            this.selectedNode = node;
                            if (node && node.data && node.data.type === 'provider') {
                                this.openProviderDetailModal(node);
                            }
                        }
                    });
                },


                // Helpers
                formatPackageName(name) {
                    if (!name) return '(anonymous)';
                    const parts = name.split('/');
                    if (parts.length > 2) {
                        return '.../' + parts.slice(-2).join('/');
                    }
                    return name;
                },

                formatTypeName(typeName) {
                    if (!typeName) return '';
                    let t = typeName.replace(/^\*/, '').replace(/^\[\]/, '');
                    const lastSlash = t.lastIndexOf('/');
                    if (lastSlash > -1) {
                        t = t.substring(lastSlash + 1);
                    }
                    if (t.length > 30) {
                        t = '...' + t.slice(-27);
                    }
                    return t;
                },

                formatFunctionName(fnName) {
                    if (!fnName) return 'unknown';
                    const lastSlash = fnName.lastIndexOf('/');
                    let name = lastSlash > -1 ? fnName.substring(lastSlash + 1) : fnName;
                    if (name.length > 35) {
                        name = '...' + name.slice(-32);
                    }
                    return name;
                },

                providerOutputTypes(provider) {
                    if (!provider || typeof provider !== 'object') {
                        return [];
                    }
                    const fromList = Array.isArray(provider.output_types)
                        ? provider.output_types.map(v => String(v || '').trim()).filter(v => v.length > 0)
                        : [];
                    if (fromList.length > 0) {
                        return fromList;
                    }
                    const fallback = String(provider.output_type || '').trim();
                    return fallback ? [fallback] : [];
                },

                primaryOutputType(provider) {
                    const list = this.providerOutputTypes(provider);
                    return list.length > 0 ? list[0] : '';
                },

                formatDurationNs(v) {
                    const ns = Number(v || 0);
                    if (!Number.isFinite(ns) || ns <= 0) return '0 ms';
                    if (ns < 1000) return `${Math.round(ns)} ns`;
                    if (ns < 1000000) return `${(ns / 1000).toFixed(2)} µs`;
                    if (ns < 1000000000) return `${(ns / 1000000).toFixed(2)} ms`;
                    return `${(ns / 1000000000).toFixed(3)} s`;
                },

                formatErrorTime(unixNano) {
                    const ns = Number(unixNano || 0);
                    if (!Number.isFinite(ns) || ns <= 0) return '-';
                    return new Date(Math.floor(ns / 1000000)).toLocaleString('zh-CN', { hour12: false });
                },

                runtimeStatKey(item, idx) {
                    return `${item.function_name || 'unknown'}|${item.output_type || ''}|${idx}`;
                },

                getRuntimeStatForProvider(provider) {
                    if (!provider || !this.runtimeStats || this.runtimeStats.length === 0) {
                        return null;
                    }
                    const exact = this.runtimeStats.find(s =>
                        s.function_name === provider.function_name && s.output_type === provider.output_type
                    );
                    if (exact) return exact;
                    return this.runtimeStats.find(s => s.function_name === provider.function_name) || null;
                },

                isTimeoutRuntimeStat(stat) {
                    if (!stat) {
                        return false;
                    }
                    const err = String(stat.last_error || '').toLowerCase();
                    return err.includes('timeout') || err.includes('deadline exceeded');
                },

                isProviderTimedOut(provider) {
                    return this.isTimeoutRuntimeStat(this.getRuntimeStatForProvider(provider));
                },

                normalizeProviderKey(v) {
                    return String(v || '').trim().toLowerCase();
                },

                getProviderErrorFor(provider) {
                    if (!provider || !this.recentErrors || this.recentErrors.length === 0) {
                        return null;
                    }
                    const fn = this.normalizeProviderKey(provider.function_name);
                    const out = this.normalizeProviderKey(provider.output_type);
                    if (!fn && !out) {
                        return null;
                    }

                    for (const err of this.recentErrors) {
                        if (!err) continue;
                        const errFn = this.normalizeProviderKey(err.provider_function);
                        const errOut = this.normalizeProviderKey(err.output_type);
                        if (fn && errFn && fn === errFn) {
                            return err;
                        }
                        if (fn && out && errFn && errOut && fn === errFn && out === errOut) {
                            return err;
                        }
                    }
                    return null;
                },

                isProviderErrored(provider) {
                    return !!this.getProviderErrorFor(provider);
                },

                providerNodeLabel(provider) {
                    const base = this.formatFunctionName(provider.function_name);
                    if (this.isProviderErrored(provider)) {
                        return `🚨 ${base}`;
                    }
                    return base;
                },

                providerErrorSummary(provider) {
                    const err = this.getProviderErrorFor(provider);
                    if (!err) {
                        return '';
                    }
                    const msg = String(err.message || '').trim();
                    const stage = String(err.stage || '').trim();
                    if (msg && stage) {
                        return `最近错误（${stage}）：${msg}`;
                    }
                    if (msg) {
                        return `最近错误：${msg}`;
                    }
                    return '最近有错误事件';
                },

                getProviderNodeColor(provider) {
                    if (this.isProviderErrored(provider)) {
                        return { background: '#fee2e2', border: '#b91c1c' };
                    }
                    if (this.isProviderTimedOut(provider)) {
                        return { background: '#fecaca', border: '#dc2626' };
                    }
                    return { background: '#bbf7d0', border: '#22c55e' };
                },

                findProviderByRuntimeStat(stat) {
                    return (this.allData?.providers || []).find(p =>
                        p.function_name === stat.function_name && p.output_type === stat.output_type
                    ) || (this.allData?.providers || []).find(p =>
                        p.function_name === stat.function_name && this.providerOutputTypes(p).includes(stat.output_type)
                    ) || (this.allData?.providers || []).find(p => p.function_name === stat.function_name) || null;
                },

                async focusProviderByRuntime(stat) {
                    if (!stat || !this.allData) return;

                    let provider = this.findProviderByRuntimeStat(stat);

                    // 如果当前按包过滤导致 provider 不在当前图里，自动切回“全部”再定位。
                    if (!provider && this.currentPackage) {
                        this.currentPackage = null;
                        await this.loadDependencies();
                        provider = this.findProviderByRuntimeStat(stat);
                    }

                    if (!provider) {
                        this.runtimeStatsError = '当前图中未找到该 provider，请清除过滤后重试';
                        this.selectedNode = {
                            data: {
                                type: 'provider',
                                function_name: stat.function_name,
                                output_type: stat.output_type,
                                input_types: []
                            }
                        };
                        return;
                    }

                    // 先展示详情，避免后续聚焦失败时右侧空白。
                    this.selectedNode = { id: provider.id, data: { ...provider, type: 'provider' } };

                    this.currentView = 'providers';
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.focusedGroup = null;
                    this.renderGraph();

                    setTimeout(() => {
                        if (!this.network || !this.network.body || !this.network.body.data || !this.network.body.data.nodes) {
                            return;
                        }

                        const targetNode = this.network.body.data.nodes.get(provider.id);
                        if (!targetNode) {
                            // 可能被分组聚合隐藏，降级到类型依赖视图，帮助用户定位。
                            this.showDependencyGraph(provider.output_type, 'type');
                            return;
                        }

                        try {
                            this.network.selectNodes([provider.id]);
                            this.network.focus(provider.id, { scale: 1.15, animation: true });
                            this.selectedNode = targetNode;
                        } catch (e) {
                            // ignore focus errors
                        }
                    }, 120);
                },

                buildProviderTooltip(provider) {
                    let tip = '函数: ' + provider.function_name + '\n';
                    const outputTypes = this.providerOutputTypes(provider);
                    if (outputTypes.length <= 1) {
                        tip += '输出: ' + (outputTypes[0] || '-') + '\n';
                    } else {
                        tip += `输出(${outputTypes.length}):\n`;
                        outputTypes.forEach((t, i) => {
                            tip += '  ' + (i + 1) + '. ' + t + '\n';
                        });
                    }
                    if (provider.input_types && provider.input_types.length > 0) {
                        tip += '输入:\n';
                        provider.input_types.forEach((t, i) => {
                            tip += '  ' + (i + 1) + '. ' + t + '\n';
                        });
                    }

                    const runtime = this.getRuntimeStatForProvider(provider);
                    if (runtime) {
                        tip += '耗时: ' + this.formatDurationNs(runtime.total_duration) + '\n';
                        if (this.isTimeoutRuntimeStat(runtime)) {
                            tip += '⚠ 超时: ' + (runtime.last_error || 'timeout') + '\n';
                        }
                    }

                    const err = this.getProviderErrorFor(provider);
                    if (err) {
                        tip += '🚨 最近错误: ' + (err.message || '-') + '\n';
                        if (err.stage) {
                            tip += '阶段: ' + err.stage + '\n';
                        }
                        if (err.error_type) {
                            tip += '类型: ' + err.error_type + '\n';
                        }
                    }

                    return tip;
                },

                addGroup() {
                    const name = (this.newGroupName || '').trim();
                    const prefix = (this.newGroupPrefix || '').trim();
                    if (!name || !prefix) return;
                    const exists = this.groupRules.find(g => g.name === name);
                    if (exists) {
                        if (!exists.prefixes.includes(prefix)) {
                            exists.prefixes.push(prefix);
                        }
                    } else {
                        this.groupRules.push({ name, prefixes: [prefix], _newPrefix: '', _rename: '' });
                    }
                    this.newGroupName = '';
                    this.newGroupPrefix = '';
                    this.saveLocalState();
                    this.renderGraph();
                },

                removeGroup(index) {
                    this.groupRules.splice(index, 1);
                    this.saveLocalState();
                    this.renderGraph();
                },

                renameGroup(groupIndex) {
                    const grp = this.groupRules[groupIndex];
                    if (!grp) return;
                    const nextName = (grp._rename || '').trim();
                    if (!nextName || nextName === grp.name) return;
                    if (this.groupRules.some((g, i) => i !== groupIndex && g.name === nextName)) {
                        return;
                    }
                    grp.name = nextName;
                    grp._rename = '';
                    this.saveLocalState();
                    this.renderGraph();
                },

                addPrefix(groupIndex) {
                    const grp = this.groupRules[groupIndex];
                    if (!grp) return;
                    const prefix = (grp._newPrefix || '').trim();
                    if (!prefix) return;
                    if (!grp.prefixes.includes(prefix)) {
                        grp.prefixes.push(prefix);
                    }
                    grp._newPrefix = '';
                    this.saveLocalState();
                    this.renderGraph();
                },

                removePrefix(groupIndex, prefixIndex) {
                    const grp = this.groupRules[groupIndex];
                    if (!grp) return;
                    grp.prefixes.splice(prefixIndex, 1);
                    this.saveLocalState();
                    this.renderGraph();
                },

                loadLocalState() {
                    try {
                        const raw = localStorage.getItem(this.storageKey);
                        if (!raw) return;
                        const data = JSON.parse(raw);
                        if (typeof data.aggregateGroups === 'boolean') {
                            this.aggregateGroups = data.aggregateGroups;
                        }
                        if (Array.isArray(data.groupRules)) {
                            this.groupRules = data.groupRules.map(g => ({
                                name: g.name,
                                prefixes: Array.isArray(g.prefixes) ? g.prefixes : [],
                                _newPrefix: '',
                                _rename: ''
                            }));
                        }
                    } catch (e) {
                        console.warn('[dix] failed to load group rules from storage', e);
                    }
                },

                saveLocalState() {
                    try {
                        const payload = {
                            aggregateGroups: this.aggregateGroups,
                            groupRules: this.groupRules.map(g => ({
                                name: g.name,
                                prefixes: g.prefixes || []
                            }))
                        };
                        localStorage.setItem(this.storageKey, JSON.stringify(payload));
                    } catch (e) {
                        console.warn('[dix] failed to save group rules to storage', e);
                    }
                },

                aggregateByGroups(nodes, edges, protectedIds = new Set()) {
                    if (!this.aggregateGroups || this.groupRules.length === 0) {
                        this.groupMembers = {};
                        return { nodes, edges };
                    }

                    const mapping = new Map();
                    const groupNodes = new Map();
                    const debugStats = new Map();
                    let typeNodeCount = 0;
                    let emptyPkgCount = 0;
                    const pkgSamples = [];
                    const members = {};

                    const makeGroup = (name) => ({
                        id: `group:${name}`,
                        label: name,
                        color: { background: '#e5e7eb', border: '#6b7280' },
                        data: { type: 'group', group: name }
                    });

                    nodes.forEach(n => {
                        if (protectedIds.has(n.id)) return;
                        const isTypeNode = n.data && n.data.type === 'type';
                        const isProviderNode = n.data && n.data.type === 'provider';
                        if (!isTypeNode && !isProviderNode) return;
                        typeNodeCount += 1;
                        if (!n.data.packagePath) {
                            emptyPkgCount += 1;
                            if (pkgSamples.length < 20) {
                                pkgSamples.push({ id: n.id, packagePath: n.data.packagePath || '' });
                            }
                        }
                        const groupName = this.matchGroup(n.id, n.data && n.data.packagePath ? n.data.packagePath : '');
                        if (!groupName) return;
                        if (this.isGroupExpanded(groupName)) return;
                        mapping.set(n.id, `group:${groupName}`);
                        if (!groupNodes.has(groupName)) {
                            groupNodes.set(groupName, makeGroup(groupName));
                        }

                        if (!members[groupName]) {
                            members[groupName] = [];
                        }
                        if (members[groupName].length < 200) {
                            members[groupName].push({
                                id: n.id,
                                packagePath: n.data.packagePath || '',
                                nodeType: n.data.type
                            });
                        }

                        if (this.debugGroupMatching) {
                            const pkg = n.data && n.data.packagePath ? n.data.packagePath : '';
                            if (!debugStats.has(groupName)) {
                                debugStats.set(groupName, []);
                            }
                            if (debugStats.get(groupName).length < 50) {
                                debugStats.get(groupName).push({ id: n.id, packagePath: pkg });
                            }
                        }
                    });

                    if (this.debugGroupMatching) {
                        console.group('[dix] group matching');
                        console.log('rules', JSON.parse(JSON.stringify(this.groupRules)));
                        console.log('type nodes', typeNodeCount, 'empty packagePath', emptyPkgCount, 'samples', pkgSamples);
                        this.groupRules.forEach(grp => {
                            const items = debugStats.get(grp.name) || [];
                            console.log(`group=${grp.name} matched=${items.length}`, items);
                        });
                        console.groupEnd();
                    }

                    this.groupMembers = members;

                    if (mapping.size === 0) {
                        return { nodes, edges };
                    }

                    const mappedNodes = nodes.filter(n => !mapping.has(n.id));
                    groupNodes.forEach(n => mappedNodes.push(n));

                    const edgeMap = new Map();
                    edges.forEach(e => {
                        const from = mapping.get(e.from) || e.from;
                        const to = mapping.get(e.to) || e.to;
                        if (from === to) return;
                        const key = `${from}=>${to}`;
                        if (edgeMap.has(key)) return;
                        edgeMap.set(key, { ...e, from, to });
                    });

                    return { nodes: mappedNodes, edges: Array.from(edgeMap.values()) };
                },

                matchGroup(typeName, pkgPathOverride = '') {
                    if (!typeName) return null;
                    const normalized = this.normalizeTypeForMatch(typeName);
                    const pkgPath = pkgPathOverride || this.extractPackagePath(normalized);
                    for (const grp of this.groupRules) {
                        for (const prefix of (grp.prefixes || [])) {
                            const p = String(prefix || '').trim();
                            if (!p) continue;
                            const target = pkgPath || normalized;
                            if (this.isPathLikePrefix(p)) {
                                if (target.startsWith(p) || target.includes(p)) {
                                    return grp.name;
                                }
                                continue;
                            }
                            if (target.includes(p)) {
                                return grp.name;
                            }
                        }
                    }
                    return null;
                },

                isPathLikePrefix(prefix) {
                    return prefix.includes('/') || prefix.startsWith('github.com/') || prefix.startsWith('gitee.com/') || prefix.startsWith('gitlab.com/');
                },

                normalizeTypeForMatch(typeName) {
                    let t = String(typeName).trim();
                    t = t.replace(/^\*/, '').replace(/^\[\]/, '');
                    if (t.startsWith('map[')) {
                        const idx = t.indexOf(']');
                        if (idx > -1 && idx < t.length - 1) {
                            t = t.slice(idx + 1);
                            t = t.replace(/^\*/, '');
                        }
                    }
                    return t;
                },

                extractPackagePath(typeName) {
                    const t = String(typeName || '').trim();
                    if (!t) return '';
                    const lastSlash = t.lastIndexOf('/');
                    const lastDot = t.lastIndexOf('.');
                    if (lastDot > -1 && lastDot > lastSlash) {
                        return t.slice(0, lastDot);
                    }
                    return '';
                },

                buildTypePkgMap() {
                    const map = new Map();
                    if (!this.allData || !this.allData.providers) return map;
                    this.allData.providers.forEach(p => {
                        if (p.output_type) {
                            map.set(p.output_type, p.output_pkg || '');
                        }
                        if (p.input_types && p.input_types.length) {
                            p.input_types.forEach((t, i) => {
                                const pkg = (p.input_pkgs && p.input_pkgs[i]) ? p.input_pkgs[i] : '';
                                if (!map.has(t)) {
                                    map.set(t, pkg);
                                }
                            });
                        }
                    });
                    return map;
                },

                getPackagePrefixSuggestions() {
                    if (!this.allData || !this.allData.providers) return [];
                    const set = new Set();
                    this.allData.providers.forEach(p => {
                        if (p.output_pkg) set.add(p.output_pkg);
                        if (p.function_pkg) set.add(p.function_pkg);
                        if (p.input_pkgs && p.input_pkgs.length) {
                            p.input_pkgs.forEach(pkg => {
                                if (pkg) set.add(pkg);
                            });
                        }
                    });
                    return Array.from(set).sort();
                },

                async exportSvg() {
                    try {
                        if (!this.network || !this.network.body) return;
                        const nodesData = this.network.body.data.nodes.get();
                        const edgesData = this.network.body.data.edges.get();
                        if (!nodesData || nodesData.length === 0) return;

                        const positions = this.network.getPositions(nodesData.map(n => n.id));
                        let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;

                        nodesData.forEach(n => {
                            const id = n.id;
                            let box;
                            try {
                                box = this.network.getBoundingBox(id);
                            } catch (e) {
                                box = null;
                            }
                            const pos = positions[id] || { x: 0, y: 0 };
                            const left = box ? box.left : pos.x - 50;
                            const right = box ? box.right : pos.x + 50;
                            const top = box ? box.top : pos.y - 25;
                            const bottom = box ? box.bottom : pos.y + 25;
                            minX = Math.min(minX, left);
                            minY = Math.min(minY, top);
                            maxX = Math.max(maxX, right);
                            maxY = Math.max(maxY, bottom);
                        });

                        if (!isFinite(minX) || !isFinite(minY) || !isFinite(maxX) || !isFinite(maxY)) return;

                        const padding = 20;
                        const width = Math.max(1, Math.round(maxX - minX + padding * 2));
                        const height = Math.max(1, Math.round(maxY - minY + padding * 2));

                        const mapX = (x) => x - minX + padding;
                        const mapY = (y) => y - minY + padding;

                        const escapeXml = (s) => String(s || '')
                            .replace(/&/g, '&amp;')
                            .replace(/</g, '&lt;')
                            .replace(/>/g, '&gt;')
                            .replace(/"/g, '&quot;')
                            .replace(/'/g, '&#39;');

                        const parts = [];
                        parts.push(`<?xml version="1.0" encoding="UTF-8"?>`);
                        parts.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">`);
                        parts.push(`<defs>`);
                        parts.push(`<marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">`);
                        parts.push(`<path d="M0,0 L0,6 L9,3 z" fill="#9ca3af" />`);
                        parts.push(`</marker>`);
                        parts.push(`</defs>`);

                        edgesData.forEach(e => {
                            const from = positions[e.from];
                            const to = positions[e.to];
                            if (!from || !to) return;
                            const x1 = mapX(from.x);
                            const y1 = mapY(from.y);
                            const x2 = mapX(to.x);
                            const y2 = mapY(to.y);
                            const color = (e.color && e.color.color) ? e.color.color : '#9ca3af';
                            const dash = e.dashes ? ' stroke-dasharray="6 4"' : '';
                            parts.push(`<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${color}" stroke-width="1.5" marker-end="url(#arrow)"${dash} />`);
                        });

                        nodesData.forEach(n => {
                            const id = n.id;
                            const pos = positions[id] || { x: 0, y: 0 };
                            let box;
                            try {
                                box = this.network.getBoundingBox(id);
                            } catch (e) {
                                box = null;
                            }
                            const left = box ? box.left : pos.x - 50;
                            const right = box ? box.right : pos.x + 50;
                            const top = box ? box.top : pos.y - 25;
                            const bottom = box ? box.bottom : pos.y + 25;
                            const w = Math.max(10, right - left);
                            const h = Math.max(10, bottom - top);
                            const x = mapX(left);
                            const y = mapY(top);
                            const cx = mapX(pos.x);
                            const cy = mapY(pos.y);

                            const bg = n.color && n.color.background ? n.color.background : '#bfdbfe';
                            const border = n.color && n.color.border ? n.color.border : '#3b82f6';
                            const fontSize = (n.font && n.font.size) ? n.font.size : 12;
                            const fontColor = (n.font && n.font.color) ? n.font.color : '#111827';

                            const nodeType = n.data && n.data.type ? n.data.type : '';
                            const isBox = n.shape === 'box' || nodeType === 'provider' || nodeType === 'group';

                            if (isBox) {
                                const rx = nodeType === 'group' ? 8 : 4;
                                parts.push(`<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${rx}" ry="${rx}" fill="${bg}" stroke="${border}" stroke-width="2" />`);
                            } else {
                                parts.push(`<ellipse cx="${cx}" cy="${cy}" rx="${w / 2}" ry="${h / 2}" fill="${bg}" stroke="${border}" stroke-width="2" />`);
                            }

                            const label = escapeXml(n.label || '');
                            if (label) {
                                parts.push(`<text x="${cx}" y="${cy}" text-anchor="middle" dominant-baseline="central" font-size="${fontSize}" font-family="system-ui, sans-serif" fill="${fontColor}">${label}</text>`);
                            }
                        });

                        parts.push(`</svg>`);

                        const svg = parts.join('\n');
                        const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' });
                        const url = URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = `dix-graph-${Date.now()}.svg`;
                        document.body.appendChild(a);
                        a.click();
                        a.remove();
                        setTimeout(() => URL.revokeObjectURL(url), 1000);
                    } catch (e) {
                        console.warn('导出 SVG 失败:', e);
                    }
                },

                openMermaid() {
                    this.mermaidError = '';
                    this.mermaidSource = this.buildMermaidSource();
                    this.mermaidOpen = true;
                    this.renderMermaid();
                },

                buildMermaidSource() {
                    const data = this.lastGraphData;
                    if (!data || !data.nodes || !data.edges) {
                        return 'flowchart TD\n  A[No data]';
                    }

                    const nodes = typeof data.nodes.get === 'function' ? data.nodes.get() : data.nodes;
                    const edges = typeof data.edges.get === 'function' ? data.edges.get() : data.edges;
                    const idMap = new Map();
                    const lines = ['flowchart TD'];

                    (nodes || []).forEach((n, idx) => {
                        const safeId = `N${idx}`;
                        idMap.set(n.id, safeId);
                        const label = this.escapeMermaidLabel(n.label || n.id);
                        lines.push(`  ${safeId}["${label}"]`);
                    });

                    (edges || []).forEach(e => {
                        const from = idMap.get(e.from);
                        const to = idMap.get(e.to);
                        if (!from || !to) return;
                        lines.push(`  ${from} --> ${to}`);
                    });

                    return lines.join('\n');
                },

                escapeMermaidLabel(label) {
                    return String(label || '')
                        .replace(/"/g, "'")
                        .replace(/\n/g, ' ')
                        .replace(/\r/g, ' ');
                },

                async renderMermaid() {
                    this.mermaidError = '';
                    const source = (this.mermaidSource || '').trim();
                    if (!source) {
                        this.mermaidSvg = '';
                        return;
                    }
                    if (!window.mermaid || !window.mermaid.render) {
                        this.mermaidError = 'Mermaid 脚本未加载';
                        return;
                    }
                    try {
                        const id = `dix-mermaid-${Date.now()}`;
                        const res = await window.mermaid.render(id, source);
                        this.mermaidSvg = res.svg || '';
                    } catch (e) {
                        this.mermaidError = 'Mermaid 渲染失败';
                    }
                },

                async copyMermaidSource() {
                    try {
                        if (navigator.clipboard && navigator.clipboard.writeText) {
                            await navigator.clipboard.writeText(this.mermaidSource || '');
                        }
                    } catch (e) {
                        console.warn('复制失败:', e);
                    }
                },



                filterByPrefix(nodes, edges, protectedIds = new Set()) {
                    const prefix = (this.filterPrefix || '').trim();
                    if (!prefix) {
                        return { nodes, edges };
                    }

                    const matches = (node) => {
                        if (!node || !node.data) return false;
                        const pkg = (node.data.packagePath || '').toString();
                        const id = (node.id || '').toString();
                        const fullType = (node.data.fullType || '').toString();
                        const fn = (node.data.function_name || '').toString();
                        return pkg.includes(prefix) || id.includes(prefix) || fullType.includes(prefix) || fn.includes(prefix);
                    };

                    const keep = new Set();
                    nodes.forEach(n => {
                        if (protectedIds.has(n.id) || matches(n)) {
                            keep.add(n.id);
                        }
                    });

                    const filteredNodes = nodes.filter(n => keep.has(n.id));
                    const filteredEdges = edges.filter(e => keep.has(e.from) && keep.has(e.to));
                    return { nodes: filteredNodes, edges: filteredEdges };
                },
            };
        }
