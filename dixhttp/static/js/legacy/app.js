const API_BASE = window.DIX_BASE || ""; // 由 template.html 内联注入(服务端替换 basePath)
        const apiUrl = (path) => (API_BASE ? API_BASE + path : path);

        function app() {
            return {
                // State
                loading: true,
                sidebarCollapsed: true,
                /** Right inspector: inventory vs node detail. */
                rightPanelCollapsed: false,
                rightPanelTab: 'inventory',
                toolbarMoreOpen: false,
                packageGroupOpen: {},
                /** Compact pyramid status for toolbar (not a floating tip). */
                pyramidStatus: '',
                stats: { provider_count: 0, object_count: 0, package_count: 0, edge_count: 0 },
                packages: [],
                uiVersion: 'arch-layout-v1',
                packageSearch: '',
                globalSearch: '',
                searchResults: [],
                currentPackage: null,
                currentView: 'providers',
                currentLayout: 'hierarchical',
                currentDepth: '2',
                /** Max pyramid level observed for current architecture view (depth UI). */
                pyramidMaxLevel: 4,
                allData: null,
                modulesData: null,
                selectedNode: null,
                network: null,
                focusedType: null,
                focusedGraph: null,
                focusedGroup: null,
                /** Provider id seed for pyramid neighborhood (upstream/downstream). */
                pyramidFocusProviderId: null,
                /** Type id seed for types pyramid neighborhood. */
                pyramidFocusTypeId: null,
                /** Hidden canvas seeds shared across architecture views (session + URL). */
                hiddenSeeds: [],
                hiddenSeedsKey: 'dix.hiddenNodeSeeds.v1',
                /** Snapshots of hiddenSeeds before each hide/clear (newest last); cap 20. */
                hideHistory: [],
                hideHistoryMax: 20,
                /** Right-click menu on graph nodes. */
                graphContextMenu: null,
                /** Transient undo toast after hide. */
                hideToast: null,
                _hideToastTimer: null,
                hiddenListOpen: false,
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
                packageSummary: { providers: [], types: [] },
                inventorySearch: '',
                edgeDeclutter: true,
                densityWarning: {
                    show: false,
                    message: '',
                    hubs: [],
                    suggestModules: false,
                    suggestAggregate: false,
                    suggestKeepBusiness: false,
                    edgesDropped: 0,
                    expanded: false,
                    minimized: false,
                },
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

                get packageGroups() {
                    const order = ['app', 'bootstrap', 'router', 'domain', 'infra', 'plugins', 'other'];
                    const labels = {
                        app: '应用',
                        bootstrap: '装配',
                        router: '路由',
                        domain: '领域',
                        infra: '基础设施',
                        plugins: '插件',
                        other: '其它',
                    };
                    const buckets = Object.fromEntries(order.map((k) => [k, []]));
                    for (const p of this.filteredPackages) {
                        const key = this.packageBucket(p.name);
                        if (!buckets[key]) buckets[key] = [];
                        buckets[key].push(p);
                    }
                    return order
                        .filter((k) => (buckets[k] || []).length)
                        .map((k) => ({
                            key: k,
                            label: labels[k] || k,
                            packages: buckets[k],
                            count: buckets[k].reduce((sum, p) => sum + (p.provider_count || 0), 0),
                        }));
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
                graphHelpers() {
                    return window.DIXGraphState || null;
                },

                async init() {
                    this.loadLocalState();
                    this.loadHiddenSeeds();
                    this.readArchitectureUrlState();
                    await this.loadGroupRules();
                    await this.loadStats();
                    await this.loadPackages();
                    await this.loadDependencies();
                    await this.loadRuntimeStats();
                    await this.loadRecentErrors();
                    await this.loadTraceRecords();
                    this.syncArchitectureUrl();
                    if (window.mermaid && window.mermaid.initialize) {
                        window.mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });
                    }
                    document.addEventListener('fullscreenchange', () => this.handleFullscreenChange());
                    this._onArchKeydown = (e) => this.handleArchitectureHotkeys(e);
                    document.addEventListener('keydown', this._onArchKeydown);
                    if (typeof this.$watch === 'function') {
                        this.$watch('selectedNode', (n) => {
                            if (n && (n.id || (n.data && n.data.function_name))) {
                                this.rightPanelTab = 'detail';
                                this.rightPanelCollapsed = false;
                            }
                        });
                    }
                },

                handleArchitectureHotkeys(e) {
                    const tag = (e.target && e.target.tagName) ? e.target.tagName.toLowerCase() : '';
                    if (tag === 'input' || tag === 'textarea' || tag === 'select' || (e.target && e.target.isContentEditable)) {
                        return;
                    }
                    if (e.key === 'Escape') {
                        this.clearHidePreview();
                        this.graphContextMenu = null;
                        return;
                    }
                    if (e.key === 'h' || e.key === 'H') {
                        if (this.selectedNode && this.selectedNode.id) {
                            e.preventDefault();
                            this.hideSelectedNodeAndDownstream();
                        }
                        return;
                    }
                    if (e.key === 'u' || e.key === 'U') {
                        if (this.hideHistory.length) {
                            e.preventDefault();
                            this.undoLastHide();
                        }
                        return;
                    }
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
                    this.pyramidFocusProviderId = null;
                    this.pyramidFocusTypeId = null;
                    this.filterPrefix = '';
                    if (pkgName) this.sidebarCollapsed = false;
                    if (this.currentView === 'modules' && pkgName) {
                        this.currentView = 'providers';
                    }
                    this.syncArchitectureUrl();
                    await this.loadDependencies();
                },

                async loadDependencies() {
                    this.loading = true;
                    try {
                        // Always fetch the full provider set so package/provider scope can
                        // include related upstream/downstream outside the selected package.
                        const url = apiUrl('/api/dependencies');
                        const res = await fetch(url);
                        this.allData = await res.json();
                        this.renderGraph();
                    } catch (e) {
                        console.error('加载依赖失败:', e);
                    } finally {
                        this.loading = false;
                    }
                },

                async switchView(view) {
                    this.currentView = view;
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.filterPrefix = '';
                    if (view === 'modules') {
                        await this.loadModulesData();
                    }
                    this.renderGraph();
                },

                async loadModulesData() {
                    try {
                        const res = await fetch(apiUrl('/api/modules'));
                        this.modulesData = await res.json();
                    } catch (e) {
                        console.error('加载模块地图失败:', e);
                        this.modulesData = [];
                    }
                    // Need providers to disperse fat packages in the module tree.
                    if (!this.allData) {
                        try {
                            const res = await fetch(apiUrl('/api/dependencies'));
                            this.allData = await res.json();
                        } catch (e) {
                            console.error('加载依赖失败:', e);
                        }
                    }
                },

                resetView() {
                    this.currentPackage = null;
                    this.pyramidFocusProviderId = null;
                    this.pyramidFocusTypeId = null;
                    this.currentView = 'providers';
                    this.packageSearch = '';
                    this.globalSearch = '';
                    this.searchResults = [];
                    this.selectedNode = null;
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.filterPrefix = '';
                    this.saveLocalState();
                    this.syncArchitectureUrl();
                    this.loadDependencies();
                },

                onDepthChange() {
                    this.normalizeCurrentDepth(2);
                    this.syncArchitectureUrl();
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
                    if (depth > 128) {
                        depth = 128;
                    }
                    const max = Number(this.pyramidMaxLevel) || 0;
                    if (depth > 0 && max > 0 && depth > max) {
                        depth = max;
                    }
                    return depth;
                },

                depthSelectOptions() {
                    const max = Math.max(1, Number(this.pyramidMaxLevel) || 1);
                    const preferred = [1, 2, 3, 5, 10];
                    const values = [...new Set([
                        ...preferred.filter((v) => v <= max),
                        max,
                    ])].sort((a, b) => a - b);
                    return [
                        ...values.map((v) => ({ value: String(v), label: `前 ${v} 级` })),
                        { value: '0', label: '全部层级' },
                    ];
                },

                updatePyramidMaxLevel(levels) {
                    if (!levels || typeof levels.values !== 'function') return;
                    let max = 1;
                    for (const lv of levels.values()) {
                        const n = Number(lv) || 1;
                        if (n > max) max = n;
                    }
                    this.pyramidMaxLevel = max;
                    const cur = Number.parseInt(String(this.currentDepth ?? ''), 10);
                    if (Number.isFinite(cur) && cur > max) {
                        this.currentDepth = String(max);
                    }
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
                showDependencyGraph(targetType, nodeType, pkgHint = '') {
                    // Types: use package-qualified pyramid neighborhood (depth 0 = unlimited).
                    // Legacy short-type BFS broke when「全部层级」parsed as maxDepth 0.
                    if (nodeType === 'type' || nodeType == null || nodeType === '') {
                        this.focusTypeNeighborhood({
                            id: String(targetType || '').includes('\x00') ? String(targetType) : '',
                            data: {
                                fullType: String(targetType || '').includes('\x00')
                                    ? String(targetType).split('\x00').pop()
                                    : String(targetType || ''),
                                packagePath: String(pkgHint || ''),
                                type: 'type',
                            },
                        });
                        return;
                    }
                    // Fallback: still support non-type callers via neighborhood on output type.
                    this.focusTypeNeighborhood({
                        data: {
                            fullType: String(targetType || ''),
                            packagePath: String(pkgHint || ''),
                            type: 'type',
                        },
                    });
                },

                /** Focus types pyramid on one type and its upstream/downstream. */
                focusTypeNeighborhood(node) {
                    if (!node) return;
                    const helpers = this.graphHelpers();
                    const data = node.data || {};
                    const fullType = String(data.fullType || data.output_type || '').trim();
                    const pkg = String(data.packagePath || data.output_pkg || '').trim();
                    let seedId = String(node.id || '').trim();
                    if ((!seedId || seedId === fullType) && helpers && helpers.typeNodeIdentity && fullType) {
                        seedId = helpers.typeNodeIdentity(fullType, pkg);
                    }
                    if (!seedId) seedId = fullType;
                    if (!seedId) return;

                    this.currentView = 'types';
                    this.pyramidFocusTypeId = seedId;
                    this.pyramidFocusProviderId = null;
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.focusedGroup = null;
                    this.rightPanelTab = 'detail';
                    this.rightPanelCollapsed = false;
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                    setTimeout(() => {
                        if (!this.network || !this.lastGraphData) return;
                        if (!this.lastGraphData.nodes.get(seedId)) return;
                        try {
                            this.network.selectNodes([seedId]);
                            this.network.focus(seedId, {
                                scale: 1.05,
                                animation: { duration: 280, easingFunction: 'easeInOutQuad' },
                            });
                            const n = this.lastGraphData.nodes.get(seedId);
                            if (n) this.selectedNode = n;
                        } catch (e) {
                            // ignore
                        }
                    }, 140);
                },

                renderModulesGraph() {
                    const helpers = this.graphHelpers();
                    if (!helpers || !helpers.buildModuleMapGraph) {
                        console.warn('[dix] graph helpers unavailable for module map');
                        return;
                    }
                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.focusedGroup = null;

                    let { nodes, edges } = helpers.buildModuleMapGraph(this.modulesData || [], {
                        providers: (this.allData && this.allData.providers) || [],
                        minDepth: 5,
                        fatThreshold: 12,
                    });
                    // Don't apply substring filterPrefix here — package tree ids are relative paths.
                    const container = document.getElementById('network');
                    nodes = nodes.map((n) => ({
                        ...n,
                        font: { ...(n.font || {}), size: 13 },
                        widthConstraint: { maximum: 150 },
                        data: { ...(n.data || {}), displayLabel: (n.data && n.data.displayLabel) || n.label },
                    }));
                    const data = {
                        nodes: new vis.DataSet(nodes),
                        edges: new vis.DataSet(edges),
                    };
                    this.lastGraphData = data;

                    // Package TREE: hierarchical by path depth; containment edges only.
                    const options = this.getNetworkOptions('hierarchical');
                    options.layout = {
                        hierarchical: {
                            enabled: true,
                            direction: 'UD',
                            sortMethod: 'directed',
                            shakeTowards: 'roots',
                            blockShifting: true,
                            edgeMinimization: true,
                            parentCentralization: true,
                            levelSeparation: 180,
                            nodeSpacing: 220,
                            treeSpacing: 240,
                        },
                    };
                    options.physics = { enabled: false };
                    options.edges = {
                        ...(options.edges || {}),
                        smooth: { type: 'cubicBezier', roundness: 0.2 },
                        width: 1.25,
                        color: { color: '#94a3b8', opacity: 0.85 },
                    };
                    options.interaction = {
                        ...(options.interaction || {}),
                        hover: true,
                        zoomView: true,
                        dragView: true,
                    };

                    if (this.network) {
                        this.network.destroy();
                    }
                    this.network = new vis.Network(container, data, options);

                    this.network.on('click', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = data.nodes.get(nodeId);
                            this.selectedNode = node;
                        }
                    });
                    this.network.on('doubleClick', params => {
                        if (params.nodes.length > 0) {
                            const nodeId = params.nodes[0];
                            const node = data.nodes.get(nodeId);
                            if (node && node.data && node.data.type === 'module') {
                                const pkg = (node.data.packagePath || node.data.module && node.data.module.name || node.id);
                                this.selectPackage(pkg);
                            }
                        }
                    });
                    this.network.on('zoom', () => this.applyLabelLod());

                    setTimeout(() => {
                        if (!this.network) return;
                        this.network.fit({ animation: false, padding: 56 });
                        // If still microscopic, focus the root instead of a hairline.
                        let scale = 1;
                        try { scale = this.network.getScale(); } catch { scale = 1; }
                        if (scale < 0.45 && nodes.length) {
                            const root = nodes.reduce((a, b) => ((a.level || 99) <= (b.level || 99) ? a : b), nodes[0]);
                            this.network.focus(root.id, { scale: 0.85, animation: false });
                        }
                        this.applyLabelLod();
                    }, 100);
                    const maxLevel = nodes.reduce((m, n) => Math.max(m, n.level || 1), 1);
                    this.densityWarning = {
                        show: true,
                        message: `模块包树：深度 ${maxLevel} · ${nodes.length} 个包节点（含路径父节点）。过肥包拆成 app/plugins/workers 等桶。双击进入 Providers。`,
                        hubs: [],
                        suggestModules: false,
                        suggestAggregate: false,
                        suggestKeepBusiness: false,
                        expanded: false,
                        minimized: true,
                    };
                },

                renderGraph() {
                    if (this.currentView === 'modules') {
                        this.renderModulesGraph();
                        return;
                    }
                    if (!this.allData) return;

                    this.focusedType = null;
                    this.focusedGraph = null;
                    this.focusedGroup = null;

                    const nodes = [];
                    const edges = [];
                    const nodeMap = new Map();
                    const typePkgMap = this.buildTypePkgMap();

                    if (this.currentView === 'providers') {
                        const helpers = this.graphHelpers();
                        const depth = this.parseDepthValue(2);
                        const seedIds = this.pyramidFocusProviderId ? [this.pyramidFocusProviderId] : [];
                        if (helpers && helpers.buildProvidersPyramidView) {
                            const built = helpers.buildProvidersPyramidView(this.allData.providers || [], {
                                depth,
                                packageName: this.currentPackage || '',
                                seedProviderIds: seedIds,
                            });
                            built.nodes.forEach((n) => {
                                const provider = n.data || {};
                                const label = this.providerNodeLabel(provider) || n.label;
                                nodes.push({
                                    ...n,
                                    label,
                                    title: this.buildProviderTooltip(provider),
                                    color: this.getProviderNodeColor(provider),
                                    shape: 'box',
                                    font: { size: 11 },
                                    level: n.level || 1,
                                    data: {
                                        ...provider,
                                        type: 'provider',
                                        packagePath: provider.output_pkg || provider.function_pkg || '',
                                        displayLabel: label,
                                        pyramidLevel: n.level || 1,
                                    },
                                });
                                nodeMap.set(n.id, true);
                            });
                            built.edges.forEach((e) => edges.push({ ...e }));
                            this._lastPyramidMeta = {
                                entries: built.entries || [],
                                depth: built.depth,
                                totalProviders: (this.allData.providers || []).length,
                            };
                            this.updatePyramidMaxLevel(built.levels);
                            if (this._lastPyramidMeta) {
                                this._lastPyramidMeta.maxLevel = this.pyramidMaxLevel;
                            }
                        } else {
                            // Fallback: provider nodes only, no type ellipses.
                            this.allData.providers.forEach((provider) => {
                                const providerNodeId = provider.id;
                                if (nodeMap.has(providerNodeId)) return;
                                const fnName = this.providerNodeLabel(provider);
                                nodes.push({
                                    id: providerNodeId,
                                    label: fnName,
                                    title: this.buildProviderTooltip(provider),
                                    color: this.getProviderNodeColor(provider),
                                    shape: 'box',
                                    font: { size: 11 },
                                    data: {
                                        ...provider,
                                        type: 'provider',
                                        packagePath: provider.output_pkg || provider.function_pkg || '',
                                    },
                                });
                                nodeMap.set(providerNodeId, true);
                            });
                            this._lastPyramidMeta = null;
                        }
                    } else if (this.currentView === 'types') {
                        const helpers = this.graphHelpers();
                        const depth = this.parseDepthValue(2);
                        const seedIds = this.pyramidFocusTypeId ? [this.pyramidFocusTypeId] : [];
                        if (helpers && helpers.buildTypesPyramidView) {
                            const built = helpers.buildTypesPyramidView(this.allData, {
                                depth,
                                packageName: this.currentPackage || '',
                                seedTypeIds: seedIds,
                            });
                            built.nodes.forEach((n) => {
                                const fullType = (n.data && n.data.fullType) || n.id;
                                const pkgPath = (n.data && n.data.packagePath) || typePkgMap.get(fullType) || '';
                                const label = (n.data && n.data.displayLabel) || n.label
                                    || (helpers.labelTypeWithPackage
                                        ? helpers.labelTypeWithPackage(fullType, pkgPath)
                                        : this.formatTypeName(fullType));
                                const isFocus = this.pyramidFocusTypeId && n.id === this.pyramidFocusTypeId;
                                nodes.push({
                                    ...n,
                                    label,
                                    title: (pkgPath ? pkgPath + '\n' : '') + fullType,
                                    color: isFocus
                                        ? { background: '#fde68a', border: '#f59e0b' }
                                        : (n.color || { background: '#bfdbfe', border: '#3b82f6' }),
                                    shape: 'ellipse',
                                    level: n.level || 1,
                                    data: {
                                        ...(n.data || {}),
                                        type: 'type',
                                        fullType,
                                        packagePath: pkgPath,
                                        displayLabel: label,
                                        pyramidLevel: n.level || 1,
                                    },
                                });
                                nodeMap.set(n.id, true);
                            });
                            built.edges.forEach((e) => edges.push({ ...e }));
                            this._lastPyramidMeta = {
                                entries: built.entries || [],
                                depth: built.depth,
                                totalTypes: built.levels ? built.levels.size : nodes.length,
                                kind: 'types',
                            };
                            this.updatePyramidMaxLevel(built.levels);
                            if (this._lastPyramidMeta) {
                                this._lastPyramidMeta.maxLevel = this.pyramidMaxLevel;
                            }
                        } else {
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
                                    from: edge.to,
                                    to: edge.from,
                                    arrows: 'to',
                                    color: { color: '#9ca3af' }
                                });
                            });
                            this._lastPyramidMeta = { kind: 'types' };
                        }
                    }

                    // Create network
                    const aggregated = this.aggregateByGroups(nodes, edges);
                    // Pyramid views already scoped by package/seeds — avoid substring cuts.
                    const filteredByPrefix = (this.currentView === 'providers' || this.currentView === 'types')
                        ? { nodes: aggregated.nodes, edges: aggregated.edges }
                        : this.filterByPrefix(aggregated.nodes, aggregated.edges);
                    let ns = filteredByPrefix.nodes;
                    let es = filteredByPrefix.edges;

                    const helpers = this.graphHelpers();
                    if (helpers && helpers.applyHiddenNodeSeeds && this.hiddenSeeds.length) {
                        const cut = helpers.applyHiddenNodeSeeds(
                            ns,
                            es,
                            this.hiddenSeeds
                        );
                        ns = cut.nodes;
                        es = cut.edges;
                        if (this.selectedNode && cut.hiddenIds.has(this.selectedNode.id)) {
                            this.selectedNode = null;
                        }
                    }
                    this.refreshPackageSummary();
                    let edgesDropped = 0;
                    if (this.edgeDeclutter && helpers && helpers.declutterEdges) {
                        const maxEdges = Math.max(200, ns.length * 3);
                        const cleaned = helpers.declutterEdges(ns, es, {
                            maxEdges,
                            hubLimit: 16,
                        });
                        if (cleaned.decluttered) {
                            es = cleaned.edges;
                            edgesDropped = cleaned.dropped || 0;
                        }
                    }

                    const cap = helpers && helpers.READABLE_NODE_CAP ? helpers.READABLE_NODE_CAP : 40;
                    const hierNodeBudget = helpers && helpers.HIERARCHICAL_NODE_BUDGET
                        ? helpers.HIERARCHICAL_NODE_BUDGET
                        : 150;
                    const hierEdgeBudget = helpers && helpers.HIERARCHICAL_EDGE_BUDGET
                        ? helpers.HIERARCHICAL_EDGE_BUDGET
                        : 400;
                    const over = ns.length > cap || es.length > cap * 3;
                    const prevWarn = this.densityWarning || {};
                    const pyramidMeta = this._lastPyramidMeta;
                    let message = '';
                    let suggestKeepBusiness = false;
                    this.pyramidStatus = '';
                    if ((this.currentView === 'providers' || this.currentView === 'types') && pyramidMeta && pyramidMeta.entries) {
                        const entryN = (pyramidMeta.entries || []).length;
                        const depthLabel = pyramidMeta.depth > 0 ? pyramidMeta.depth : '全部';
                        const kindLabel = this.currentView === 'types' ? 'Type' : 'Provider';
                        this.pyramidStatus = `${kindLabel} · ${entryN} 入口 · ${depthLabel} 层 · ${ns.length} 节点`;
                        const unscoped = !this.currentPackage && !this.pyramidFocusProviderId && !this.pyramidFocusTypeId;
                        if (entryN > 12 && unscoped) {
                            message = `入口较多（${entryN}）。可选左侧包缩小范围`;
                            if (this.currentView === 'providers' && !(this.hiddenSeeds && this.hiddenSeeds.length)) {
                                message += '，或「只留业务入口」收敛 diag / Dix / plugins。';
                                suggestKeepBusiness = true;
                            } else {
                                message += '。';
                            }
                        }
                    } else if (over) {
                        message = `图规模较大（${ns.length} 节点 / ${es.length} 边）。可先看模块地图，或缩小左侧包范围。`;
                    }
                    if (edgesDropped > 0) {
                        message += (message ? ' ' : '')
                            + `已隐藏 ${edgesDropped} 条次要边。`;
                    }
                    const actionable = !!(suggestKeepBusiness || over || edgesDropped > 0);
                    this.densityWarning = {
                        show: actionable,
                        message,
                        hubs: (over && helpers && helpers.rankHubNodes) ? helpers.rankHubNodes(ns, es, 8) : [],
                        suggestModules: over && this.currentView !== 'modules',
                        suggestAggregate: over && !this.aggregateGroups && (this.groupRules || []).length > 0,
                        suggestKeepBusiness,
                        edgesDropped,
                        expanded: false,
                        minimized: actionable ? (prevWarn.show ? !!prevWarn.minimized : true) : false,
                    };
                    if (helpers && helpers.applyGraphBudget
                        && this.currentView !== 'providers'
                        && this.currentView !== 'types'
                        && (ns.length > hierNodeBudget || es.length > hierEdgeBudget)) {
                        const beforeN = ns.length;
                        const bounded = helpers.applyGraphBudget(ns, es, {
                            nodes: hierNodeBudget,
                            edges: hierEdgeBudget,
                        });
                        ns = bounded.nodes;
                        es = bounded.edges;
                        if (bounded.degraded) {
                            this.densityWarning.show = true;
                            this.densityWarning.message = (this.densityWarning.message
                                ? this.densityWarning.message + ' '
                                : '')
                                + `为可读性保留约 ${ns.length}/${beforeN} 个高连通节点；完整列表见右侧清单或缩小包范围。`;
                            if (!prevWarn.show) this.densityWarning.minimized = true;
                        }
                    }

                    const container = document.getElementById('network');
                    const preferred = this.currentLayout === 'force' ? 'physics' : 'hierarchical';
                    const stripRisk = helpers && helpers.estimateHierarchicalStripRisk
                        ? helpers.estimateHierarchicalStripRisk(ns, es)
                        : { wide: false };
                    const effective = helpers && helpers.resolveEffectiveLayout
                        ? helpers.resolveEffectiveLayout(preferred, ns.length)
                        : preferred;
                    if ((over || stripRisk.wide) && this.currentLayout === 'hierarchical') {
                        const scopeTip = stripRisk.wide
                            ? `图过宽（约 ${stripRisk.roots || '?'} 个根/岛）。请先选左侧包范围或改用模块地图。`
                            : '';
                        if (scopeTip) {
                            this.densityWarning.show = true;
                            this.densityWarning.message = (this.densityWarning.message
                                ? this.densityWarning.message + ' '
                                : '') + scopeTip;
                            if (!this.densityWarning.hubs.length && helpers && helpers.rankHubNodes) {
                                this.densityWarning.hubs = helpers.rankHubNodes(ns, es, 8);
                            }
                            this.densityWarning.suggestModules = this.currentView !== 'modules';
                            if (!prevWarn.show) this.densityWarning.minimized = true;
                        }
                    }
                    const options = this.getNetworkOptions(effective);
                    const fontSize = ns.length <= 12 ? 15 : ns.length <= 24 ? 13 : 11;
                    ns = ns.map(n => ({
                        ...n,
                        level: n.level || (n.data && n.data.pyramidLevel) || n.level,
                        font: { ...(n.font || {}), size: Math.max((n.font && n.font.size) || 0, fontSize) },
                        data: { ...(n.data || {}), displayLabel: n.label },
                    }));
                    const data = {
                        nodes: new vis.DataSet(ns),
                        edges: new vis.DataSet(es)
                    };
                    this.lastGraphData = data;

                    if (this.network) {
                        this.clearHidePreview();
                        this.network.destroy();
                    }

                    this.network = new vis.Network(container, data, options);

                    const bindGraphInteractions = () => {
                        if (!this.network) return;
                        this.network.off('zoom');
                        this.network.off('click');
                        this.network.off('doubleClick');
                        this.network.off('oncontext');
                        this.clearHidePreview();
                        this.graphContextMenu = null;
                        const canvas = document.getElementById('network');
                        if (canvas && !canvas._dixCtxBound) {
                            canvas.addEventListener('contextmenu', (e) => e.preventDefault());
                            canvas._dixCtxBound = true;
                        }
                        this.network.on('zoom', () => this.applyLabelLod());
                        this.network.on('click', params => {
                            this.clearHidePreview();
                            this.graphContextMenu = null;
                            if (params.nodes.length > 0) {
                                const nodeId = params.nodes[0];
                                const node = data.nodes.get(nodeId);
                                // Shift+click: one-step hide (keep main graph tidy).
                                const ev = params.event && (params.event.srcEvent || params.event);
                                if (ev && ev.shiftKey) {
                                    this.hideNodeAndDownstream(node);
                                    return;
                                }
                                this.selectedNode = node;
                            } else {
                                this.selectedNode = null;
                            }
                        });
                        this.network.on('oncontext', (params) => {
                            if (params.event && params.event.preventDefault) {
                                params.event.preventDefault();
                            }
                            const dom = params.pointer && params.pointer.DOM;
                            const nodeId = dom ? this.network.getNodeAt(dom) : null;
                            if (!nodeId) {
                                this.clearHidePreview();
                                this.graphContextMenu = null;
                                return;
                            }
                            const node = data.nodes.get(nodeId);
                            if (!node) return;
                            this.selectedNode = node;
                            const count = this.estimateHiddenCount(node.id);
                            const rect = (canvas && canvas.getBoundingClientRect) ? canvas.getBoundingClientRect() : { left: 0, top: 0 };
                            this.applyHidePreview(String(node.id));
                            this.graphContextMenu = {
                                x: rect.left + (dom.x || 0),
                                y: rect.top + (dom.y || 0),
                                nodeId: String(node.id),
                                label: this.nodeDisplayLabel(node),
                                count,
                                packageOptions: this.packageHideOptions(node),
                            };
                        });
                        this.network.on('doubleClick', params => {
                            this.clearHidePreview();
                            this.graphContextMenu = null;
                            if (params.nodes.length > 0) {
                                const nodeId = params.nodes[0];
                                const node = data.nodes.get(nodeId);
                                if (node && node.data && node.data.type === 'group') {
                                    this.toggleGroupExpand(node.data.group);
                                    return;
                                }
                                if (node && node.data && node.data.type === 'module') {
                                    this.filterPrefix = (node.data.module && node.data.module.name) || node.id;
                                    this.switchView('providers');
                                    return;
                                }
                                if (node && node.data && node.data.type === 'provider') {
                                    this.pyramidFocusProviderId = node.id;
                                    this.pyramidFocusTypeId = null;
                                    this.renderGraph();
                                    return;
                                }
                                if (node && node.data && node.data.type === 'type') {
                                    this.pyramidFocusTypeId = node.id;
                                    this.pyramidFocusProviderId = null;
                                    this.currentView = 'types';
                                    this.renderGraph();
                                    return;
                                }
                                if (node && node.data) {
                                    if (node.data.output_type) {
                                        this.focusOnType(node.data.output_type);
                                    }
                                }
                            }
                        });
                    };

                    const applyReadableCamera = () => {
                        if (!this.network) return;
                        this.network.fit({ animation: false, padding: 48 });
                        let scale = 1;
                        try {
                            scale = this.network.getScale();
                        } catch {
                            scale = 1;
                        }
                        const cam = helpers && helpers.resolvePostFitCamera
                            ? helpers.resolvePostFitCamera(scale)
                            : (scale < 0.55 ? 'focus' : 'fit');
                        if (cam === 'focus') {
                            if (this.currentLayout === 'hierarchical' && ns.length > 40) {
                                const pos = this.network.getViewPosition();
                                this.network.moveTo({ scale: 0.32, position: pos, animation: false });
                                this.densityWarning.show = true;
                                this.densityWarning.message = (this.densityWarning.message
                                    ? this.densityWarning.message + ' '
                                    : '')
                                    + `层级图较宽，当前约 ${ns.length} 个节点在画布中；拖拽查看，或缩小包范围 / 改用模块地图。`;
                                if (!prevWarn.show) this.densityWarning.minimized = true;
                            } else {
                                const focusId = helpers && helpers.pickFocusNodeId
                                    ? helpers.pickFocusNodeId(ns, es, this.filterPrefix || this.focusedType || '')
                                    : (ns[0] && ns[0].id);
                                if (focusId) {
                                    this.network.focus(focusId, { scale: 1.05, animation: false });
                                }
                                if (!this.densityWarning.show) {
                                    this.densityWarning.show = true;
                                    this.densityWarning.message = '数据过多导致全图过小。请缩小包范围或改用模块地图。';
                                    this.densityWarning.suggestModules = this.currentView !== 'modules';
                                    this.densityWarning.hubs = helpers && helpers.rankHubNodes
                                        ? helpers.rankHubNodes(ns, es, 8)
                                        : [];
                                    this.densityWarning.minimized = true;
                                }
                            }
                        }
                        this.applyLabelLod();
                    };

                    bindGraphInteractions();
                    if (effective === 'hierarchical') {
                        setTimeout(applyReadableCamera, 80);
                    } else {
                        this.network.once('stabilizationIterationsDone', applyReadableCamera);
                        setTimeout(applyReadableCamera, 450);
                    }
                },

                getNetworkOptions(layoutOverride = null) {
                    const mode = layoutOverride != null
                        ? layoutOverride
                        : (this.currentLayout === 'force' ? 'physics' : 'hierarchical');
                    const isHierarchical = mode === 'hierarchical' || mode === true;
                    const usePhysics = mode === 'physics' || mode === 'force' || mode === false;
                    const architecturePyramid = (this.currentView === 'providers' || this.currentView === 'types') && isHierarchical;
                    // Boxes/ellipses need spacing wider than node width or siblings glue together.
                    const levelSeparation = architecturePyramid ? 200 : 150;
                    const nodeSpacing = architecturePyramid ? 260 : 180;
                    const treeSpacing = architecturePyramid ? 280 : 200;
                    return {
                        nodes: {
                            shape: 'box',
                            font: { size: architecturePyramid ? 13 : 12, face: 'system-ui, sans-serif' },
                            borderWidth: 2,
                            shadow: { enabled: true, size: 5, x: 2, y: 2 },
                            margin: 12,
                            widthConstraint: { maximum: architecturePyramid ? 140 : 160 },
                        },
                        edges: {
                            width: architecturePyramid ? 1 : 1.5,
                            color: architecturePyramid
                                ? { color: '#c4c9d4', opacity: 0.55, highlight: '#64748b' }
                                : undefined,
                            smooth: isHierarchical
                                ? { type: 'cubicBezier', roundness: architecturePyramid ? 0.25 : 0.4 }
                                : { type: 'continuous' }
                        },
                        physics: {
                            enabled: usePhysics && !isHierarchical,
                            stabilization: { iterations: 180 },
                            barnesHut: {
                                gravitationalConstant: -3500,
                                springLength: 180,
                                springConstant: 0.04,
                                avoidOverlap: 0.8,
                            }
                        },
                        interaction: {
                            hover: true,
                            tooltipDelay: 100,
                            zoomView: true,
                            dragView: true,
                            multiselect: false,
                        },
                        layout: isHierarchical ? {
                            hierarchical: {
                                enabled: true,
                                direction: 'UD',
                                sortMethod: 'directed',
                                shakeTowards: 'roots',
                                blockShifting: true,
                                edgeMinimization: true,
                                parentCentralization: true,
                                levelSeparation,
                                nodeSpacing,
                                treeSpacing
                            }
                        } : {
                            hierarchical: { enabled: false }
                        }
                    };
                },

                applyLabelLod() {
                    if (!this.network || !this.lastGraphData || !this.lastGraphData.nodes) return;
                    const helpers = this.graphHelpers();
                    if (!helpers || !helpers.labelLodVisibleIds) return;
                    let scale = 1;
                    try {
                        scale = this.network.getScale();
                    } catch {
                        scale = 1;
                    }
                    const nodes = this.lastGraphData.nodes.get();
                    const edges = this.lastGraphData.edges.get();
                    const visible = helpers.labelLodVisibleIds(nodes, edges, scale, { hubLimit: 12, hideBelow: 0.7 });
                    const updates = nodes.map(n => {
                        const full = (n.data && n.data.displayLabel) || n.label || '';
                        const show = visible.has(n.id);
                        return {
                            id: n.id,
                            label: show ? full : '·',
                            font: {
                                ...(n.font || {}),
                                size: show ? Math.max((n.font && n.font.size) || 12, 11) : 9,
                            },
                        };
                    });
                    this.lastGraphData.nodes.update(updates);
                },
                fitGraph() {
                    if (!this.network) return;
                    this.network.fit({ animation: { duration: 280, easingFunction: 'easeInOutQuad' }, padding: 48 });
                    setTimeout(() => this.applyLabelLod(), 300);
                },

                refreshPackageSummary() {
                    const helpers = this.graphHelpers();
                    if (!this.allData || !helpers) {
                        this.packageSummary = { providers: [], types: [] };
                        return;
                    }
                    if (helpers.buildProviderInventory) {
                        this.packageSummary = helpers.buildProviderInventory(
                            this.allData,
                            this.currentPackage || ""
                        );
                        return;
                    }
                    if (!this.currentPackage || !helpers.buildPackageSummary) {
                        this.packageSummary = { providers: [], types: [] };
                        return;
                    }
                    this.packageSummary = helpers.buildPackageSummary(this.allData, this.currentPackage);
                },

                filteredInventoryProviders() {
                    const q = String(this.inventorySearch || this.globalSearch || "").trim().toLowerCase();
                    const list = (this.packageSummary && this.packageSummary.providers) || [];
                    if (!q) return list;
                    return list.filter((p) => {
                        const hay = `${p.function_name || ""} ${p.output_type || ""} ${p.output_pkg || ""} ${p.label || ""}`.toLowerCase();
                        return hay.includes(q);
                    });
                },
                focusSummaryProvider(item) {
                    if (!item) return;
                    const provider = (this.allData?.providers || []).find((p) => p.id === item.id)
                        || (this.allData?.providers || []).find((p) => p.function_name === item.function_name);
                    if (!provider) return;
                    this.currentView = 'providers';
                    this.pyramidFocusProviderId = provider.id;
                    this.selectedNode = { id: provider.id, data: { ...provider, type: 'provider' } };
                    this.renderGraph();
                    setTimeout(() => {
                        if (!this.network || !this.lastGraphData) return;
                        const nodeId = this.lastGraphData.nodes.get(provider.id) ? provider.id : null;
                        if (!nodeId) return;
                        try {
                            this.network.selectNodes([nodeId]);
                            this.network.focus(nodeId, {
                                scale: 1.2,
                                animation: { duration: 280, easingFunction: 'easeInOutQuad' },
                            });
                            const n = this.lastGraphData.nodes.get(nodeId);
                            if (n) this.selectedNode = n;
                            setTimeout(() => this.applyLabelLod(), 300);
                        } catch (e) {
                            // ignore focus errors
                        }
                    }, 120);
                },

                focusSummaryType(item) {
                    if (!item || !item.id) return;
                    this.currentView = 'types';
                    this.pyramidFocusTypeId = item.id;
                    this.pyramidFocusProviderId = null;
                    this.renderGraph();
                    setTimeout(() => {
                        if (!this.network || !this.lastGraphData) return;
                        const typeId = item.id;
                        if (!this.lastGraphData.nodes.get(typeId)) return;
                        try {
                            this.network.selectNodes([typeId]);
                            this.network.focus(typeId, {
                                scale: 1.2,
                                animation: { duration: 280, easingFunction: 'easeInOutQuad' },
                            });
                            const n = this.lastGraphData.nodes.get(typeId);
                            if (n) this.selectedNode = n;
                            setTimeout(() => this.applyLabelLod(), 300);
                        } catch (e) {
                            // ignore
                        }
                    }, 120);
                },

                focusHubNode() {
                    if (!this.network || !this.lastGraphData) return;
                    const helpers = this.graphHelpers();
                    const nodes = this.lastGraphData.nodes.get();
                    const edges = this.lastGraphData.edges.get();
                    const focusId = helpers && helpers.pickFocusNodeId
                        ? helpers.pickFocusNodeId(nodes, edges, this.filterPrefix || this.focusedType || '')
                        : (nodes[0] && nodes[0].id);
                    if (!focusId) return;
                    this.network.focus(focusId, {
                        scale: 1.2,
                        animation: { duration: 280, easingFunction: 'easeInOutQuad' },
                    });
                    setTimeout(() => this.applyLabelLod(), 300);
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
                packageBucket(name) {
                    const parts = String(name || '').split('/').filter(Boolean);
                    for (const key of ['domain', 'plugins', 'infra', 'app', 'bootstrap', 'router']) {
                        if (parts.includes(key)) return key;
                    }
                    return 'other';
                },

                packageLeafLabel(name) {
                    const parts = String(name || '').split('/').filter(Boolean);
                    for (const key of ['domain', 'plugins', 'infra']) {
                        const i = parts.indexOf(key);
                        if (i >= 0) {
                            const rest = parts.slice(i + 1).join('/');
                            return rest || key;
                        }
                    }
                    return this.formatPackageName(name);
                },

                isPackageGroupOpen(key) {
                    if (Object.prototype.hasOwnProperty.call(this.packageGroupOpen || {}, key)) {
                        return !!this.packageGroupOpen[key];
                    }
                    return key === 'domain' || key === 'app' || key === 'plugins';
                },

                togglePackageGroup(key) {
                    const open = !this.isPackageGroupOpen(key);
                    this.packageGroupOpen = { ...(this.packageGroupOpen || {}), [key]: open };
                },

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
                    const helpers = this.graphHelpers();
                    if (helpers && helpers.shortGraphLabel) {
                        return helpers.shortGraphLabel(typeName);
                    }
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
                    const helpers = this.graphHelpers();
                    if (helpers && helpers.shortGraphLabel) {
                        return helpers.shortGraphLabel(fnName);
                    }
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
                    const helpers = this.graphHelpers();
                    const base = (helpers && helpers.providerDisplayLabel)
                        ? helpers.providerDisplayLabel(provider)
                        : (this.formatTypeName(provider.output_type)
                            || this.formatFunctionName(provider.function_name));
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
                        if (typeof data.edgeDeclutter === 'boolean') {
                            this.edgeDeclutter = data.edgeDeclutter;
                        }
                        if (typeof data.currentLayout === 'string'
                            && ['hierarchical', 'force'].includes(data.currentLayout)) {
                            this.currentLayout = data.currentLayout;
                        } else if (data.currentLayout === 'panorama') {
                            this.currentLayout = 'hierarchical';
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
                            edgeDeclutter: this.edgeDeclutter,
                            currentLayout: this.currentLayout,
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

                loadHiddenSeeds() {
                    try {
                        const raw = sessionStorage.getItem(this.hiddenSeedsKey);
                        if (!raw) {
                            this.hiddenSeeds = [];
                            return;
                        }
                        const data = JSON.parse(raw);
                        this.hiddenSeeds = Array.isArray(data)
                            ? data.filter((s) => s && (s.id || s.packagePrefix)).map((s) => ({
                                id: String(s.id || ('pkg:' + s.packagePrefix)),
                                label: String(s.label || s.packagePrefix || s.id),
                                packagePrefix: s.packagePrefix ? String(s.packagePrefix) : undefined,
                            }))
                            : [];
                    } catch (e) {
                        console.warn('[dix] failed to load hidden seeds', e);
                        this.hiddenSeeds = [];
                    }
                },

                saveHiddenSeeds() {
                    try {
                        sessionStorage.setItem(this.hiddenSeedsKey, JSON.stringify(this.hiddenSeeds));
                    } catch (e) {
                        console.warn('[dix] failed to save hidden seeds', e);
                    }
                    this.syncArchitectureUrl();
                },

                /** URL wins over session when `hide` is present; also restore depth/pkg. */
                readArchitectureUrlState() {
                    try {
                        const params = new URLSearchParams(window.location.search);
                        if (params.has('hide')) {
                            this.hiddenSeeds = this.decodeHiddenSeedsParam(params.get('hide'));
                            try {
                                sessionStorage.setItem(this.hiddenSeedsKey, JSON.stringify(this.hiddenSeeds));
                            } catch (e) {
                                // ignore
                            }
                        }
                        if (params.has('depth')) {
                            const d = String(params.get('depth') || '').trim();
                            if (d !== '') this.currentDepth = d;
                        }
                        if (params.has('pkg')) {
                            const pkg = String(params.get('pkg') || '').trim();
                            this.currentPackage = pkg || null;
                            if (pkg) this.sidebarCollapsed = false;
                        }
                    } catch (e) {
                        console.warn('[dix] failed to read architecture URL state', e);
                    }
                },

                encodeHiddenSeedsParam(seeds) {
                    const helpers = this.graphHelpers();
                    if (helpers && helpers.encodeHiddenSeedsForUrl) {
                        return helpers.encodeHiddenSeedsForUrl(seeds);
                    }
                    return (seeds || [])
                        .map((s) => {
                            if (!s) return '';
                            if (s.packagePrefix) return 'pkg:' + s.packagePrefix;
                            return String(s.id || '');
                        })
                        .filter(Boolean)
                        .join('|');
                },

                decodeHiddenSeedsParam(param) {
                    const helpers = this.graphHelpers();
                    if (helpers && helpers.decodeHiddenSeedsFromUrl) {
                        return helpers.decodeHiddenSeedsFromUrl(param);
                    }
                    if (param == null || String(param).trim() === '') return [];
                    return String(param).split('|').map((part) => {
                        const token = String(part || '').trim();
                        if (!token) return null;
                        if (token.startsWith('pkg:')) {
                            const pref = token.slice(4).trim();
                            if (!pref) return null;
                            return {
                                id: 'pkg:' + pref,
                                label: pref.split('/').filter(Boolean).slice(-2).join('/') || pref,
                                packagePrefix: pref,
                            };
                        }
                        return { id: token, label: token };
                    }).filter(Boolean);
                },

                syncArchitectureUrl() {
                    try {
                        const url = new URL(window.location.href);
                        const encoded = this.encodeHiddenSeedsParam(this.hiddenSeeds);
                        if (encoded) url.searchParams.set('hide', encoded);
                        else url.searchParams.delete('hide');

                        const depth = String(this.currentDepth ?? '').trim();
                        if (depth && depth !== '2') url.searchParams.set('depth', depth);
                        else url.searchParams.delete('depth');

                        if (this.currentPackage) url.searchParams.set('pkg', this.currentPackage);
                        else url.searchParams.delete('pkg');

                        const next = url.pathname + url.search + url.hash;
                        const cur = window.location.pathname + window.location.search + window.location.hash;
                        if (next !== cur) {
                            history.replaceState(null, '', next);
                        }
                    } catch (e) {
                        console.warn('[dix] failed to sync architecture URL', e);
                    }
                },

                cloneHiddenSeeds(seeds) {
                    return (seeds || []).map((s) => ({
                        id: String(s.id),
                        label: String(s.label || s.id),
                        packagePrefix: s.packagePrefix ? String(s.packagePrefix) : undefined,
                    }));
                },

                pushHideHistory() {
                    const snap = this.cloneHiddenSeeds(this.hiddenSeeds);
                    const next = [...(this.hideHistory || []), snap];
                    const max = this.hideHistoryMax || 20;
                    this.hideHistory = next.length > max ? next.slice(next.length - max) : next;
                },

                selectedNodeHideLabel() {
                    return this.nodeDisplayLabel(this.selectedNode);
                },

                nodeDisplayLabel(node) {
                    if (!node) return '';
                    const d = node.data || {};
                    const raw = d.displayLabel || node.label || d.function_name || d.fullType || d.group
                        || d.packagePath || node.id || '';
                    return String(raw).split('\n')[0].trim();
                },

                estimateHiddenCount(nodeId) {
                    const helpers = this.graphHelpers();
                    if (!helpers || !helpers.collectDownstreamNodeIds || !this.lastGraphData) {
                        return 1;
                    }
                    try {
                        const nodes = this.lastGraphData.nodes.get();
                        const edges = this.lastGraphData.edges.get();
                        return helpers.collectDownstreamNodeIds(nodes, edges, [String(nodeId)]).size || 1;
                    } catch (e) {
                        return 1;
                    }
                },

                estimatePackageHideCount(prefix) {
                    const helpers = this.graphHelpers();
                    if (!helpers || !this.lastGraphData) return 0;
                    try {
                        const nodes = this.lastGraphData.nodes.get();
                        const edges = this.lastGraphData.edges.get();
                        const expanded = helpers.expandHiddenSeedsToNodeIds
                            ? helpers.expandHiddenSeedsToNodeIds(nodes, [{ packagePrefix: prefix }])
                            : [];
                        if (!expanded.length) return 0;
                        return helpers.collectDownstreamNodeIds(nodes, edges, expanded).size || expanded.length;
                    } catch (e) {
                        return 0;
                    }
                },

                nodePackagePath(node) {
                    if (!node) return '';
                    const d = node.data || {};
                    if (d.type === 'module') {
                        return String(d.packagePath || node.id || '');
                    }
                    return String(d.packagePath || d.output_pkg || '').trim();
                },

                packageHideOptions(node) {
                    const pkg = this.nodePackagePath(node);
                    if (!pkg) return [];
                    const opts = [];
                    const push = (prefix, shortLabel) => {
                        if (!prefix || opts.some((o) => o.prefix === prefix)) return;
                        opts.push({
                            prefix,
                            label: shortLabel || prefix.split('/').slice(-2).join('/'),
                            count: this.estimatePackageHideCount(prefix),
                        });
                    };
                    push(pkg, pkg.split('/').filter(Boolean).slice(-2).join('/') || pkg);
                    const parts = pkg.split('/').filter(Boolean);
                    const pluginsAt = parts.lastIndexOf('plugins');
                    if (pluginsAt >= 0) {
                        push(parts.slice(0, pluginsAt + 1).join('/'), '…/plugins');
                    }
                    const diagAt = parts.lastIndexOf('diag');
                    if (diagAt >= 0) {
                        push(parts.slice(0, diagAt + 1).join('/'), '…/diag');
                    }
                    const domainAt = parts.lastIndexOf('domain');
                    if (domainAt >= 0 && parts.length > domainAt + 1) {
                        push(parts.slice(0, domainAt + 2).join('/'), '…/domain/' + parts[domainAt + 1]);
                    }
                    return opts.filter((o) => o.count > 0);
                },

                clearHidePreview() {
                    if (!this._hidePreviewBackup || !this.lastGraphData || !this.lastGraphData.nodes) {
                        this._hidePreviewBackup = null;
                        return;
                    }
                    try {
                        this.lastGraphData.nodes.update(this._hidePreviewBackup);
                    } catch (e) {
                        // network may already be destroyed
                    }
                    this._hidePreviewBackup = null;
                },

                applyHidePreview(seedId) {
                    this.clearHidePreview();
                    const helpers = this.graphHelpers();
                    if (!helpers || !helpers.collectDownstreamNodeIds || !this.lastGraphData) return;
                    const nodes = this.lastGraphData.nodes.get();
                    const edges = this.lastGraphData.edges.get();
                    const hide = helpers.collectDownstreamNodeIds(nodes, edges, [String(seedId)]);
                    if (!hide.size) return;
                    const backup = [];
                    const updates = [];
                    for (const n of nodes) {
                        backup.push({
                            id: n.id,
                            color: n.color,
                            opacity: n.opacity,
                            borderWidth: n.borderWidth,
                        });
                        if (hide.has(n.id)) {
                            updates.push({
                                id: n.id,
                                borderWidth: 3,
                                color: {
                                    background: (n.color && n.color.background) || '#fecaca',
                                    border: '#e11d48',
                                    highlight: n.color && n.color.highlight,
                                },
                                opacity: 1,
                            });
                        } else {
                            updates.push({
                                id: n.id,
                                opacity: 0.22,
                            });
                        }
                    }
                    this._hidePreviewBackup = backup;
                    try {
                        this.lastGraphData.nodes.update(updates);
                    } catch (e) {
                        this._hidePreviewBackup = null;
                    }
                },

                fitGraphCameraSoon() {
                    setTimeout(() => {
                        if (!this.network) return;
                        try {
                            this.network.fit({ animation: { duration: 220, easingFunction: 'easeInOutQuad' }, padding: 48 });
                            this.applyLabelLod();
                        } catch (e) {
                            // ignore
                        }
                    }, 100);
                },

                showHideToast(message, seedId) {
                    if (this._hideToastTimer) {
                        clearTimeout(this._hideToastTimer);
                        this._hideToastTimer = null;
                    }
                    this.hideToast = {
                        message,
                        seedId: String(seedId || ''),
                        canUndo: (this.hideHistory || []).length > 0,
                    };
                    this._hideToastTimer = setTimeout(() => {
                        this.hideToast = null;
                        this._hideToastTimer = null;
                    }, 5000);
                },

                hideNodeAndDownstream(node) {
                    if (!node || !node.id) return;
                    const id = String(node.id);
                    const label = this.nodeDisplayLabel(node) || id;
                    if (this.hiddenSeeds.some((s) => s.id === id)) {
                        this.graphContextMenu = null;
                        this.showHideToast(`「${label}」已在隐藏列表中`, id);
                        return;
                    }
                    const count = this.estimateHiddenCount(id);
                    this.pushHideHistory();
                    this.hiddenSeeds = [
                        ...this.hiddenSeeds,
                        { id, label },
                    ];
                    this.saveHiddenSeeds();
                    this.selectedNode = null;
                    this.clearHidePreview();
                    this.graphContextMenu = null;
                    this.hiddenListOpen = true;
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                    const down = Math.max(0, count - 1);
                    this.showHideToast(
                        down > 0 ? `已隐藏「${label}」及 ${down} 个下游` : `已隐藏「${label}」`,
                        id
                    );
                },

                hideSelectedNodeAndDownstream() {
                    this.hideNodeAndDownstream(this.selectedNode);
                },

                hideFromContextMenu() {
                    if (!this.graphContextMenu) return;
                    const id = this.graphContextMenu.nodeId;
                    const node = (this.lastGraphData && this.lastGraphData.nodes)
                        ? this.lastGraphData.nodes.get(id)
                        : { id, label: this.graphContextMenu.label, data: { displayLabel: this.graphContextMenu.label } };
                    this.hideNodeAndDownstream(node || { id, label: this.graphContextMenu.label });
                },

                hidePackageFromContextMenu(prefix, label) {
                    this.hidePackagePrefix(prefix, label);
                },

                hidePackagePrefix(prefix, label) {
                    const pref = String(prefix || '').trim();
                    if (!pref) return;
                    const id = 'pkg:' + pref;
                    if (this.hiddenSeeds.some((s) => s.id === id || s.packagePrefix === pref)) {
                        this.graphContextMenu = null;
                        this.clearHidePreview();
                        this.showHideToast(`「${label || pref}」已在隐藏列表中`, id);
                        return;
                    }
                    const count = this.estimatePackageHideCount(pref);
                    this.pushHideHistory();
                    this.hiddenSeeds = [
                        ...this.hiddenSeeds,
                        { id, label: label || pref.split('/').slice(-2).join('/'), packagePrefix: pref },
                    ];
                    this.saveHiddenSeeds();
                    this.selectedNode = null;
                    this.clearHidePreview();
                    this.graphContextMenu = null;
                    this.hiddenListOpen = true;
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                    this.showHideToast(
                        count > 1 ? `已隐藏包「${label || pref}」及相关 ${count} 个节点` : `已隐藏包「${label || pref}」`,
                        id
                    );
                },

                previewPackageHide(prefix) {
                    if (!prefix || !this.lastGraphData) return;
                    const helpers = this.graphHelpers();
                    if (!helpers || !helpers.expandHiddenSeedsToNodeIds) return;
                    const nodes = this.lastGraphData.nodes.get();
                    const edges = this.lastGraphData.edges.get();
                    const expanded = helpers.expandHiddenSeedsToNodeIds(nodes, [{ packagePrefix: prefix }]);
                    if (!expanded.length) return;
                    // Reuse node preview by temporarily painting expanded set
                    this.clearHidePreview();
                    const hide = helpers.collectDownstreamNodeIds(nodes, edges, expanded);
                    const backup = [];
                    const updates = [];
                    for (const n of nodes) {
                        backup.push({
                            id: n.id,
                            color: n.color,
                            opacity: n.opacity,
                            borderWidth: n.borderWidth,
                        });
                        if (hide.has(n.id)) {
                            updates.push({
                                id: n.id,
                                borderWidth: 3,
                                color: {
                                    background: (n.color && n.color.background) || '#fecaca',
                                    border: '#e11d48',
                                },
                                opacity: 1,
                            });
                        } else {
                            updates.push({ id: n.id, opacity: 0.22 });
                        }
                    }
                    this._hidePreviewBackup = backup;
                    try {
                        this.lastGraphData.nodes.update(updates);
                    } catch (e) {
                        this._hidePreviewBackup = null;
                    }
                },

                keepBusinessEntriesOnly() {
                    if (!this.lastGraphData) {
                        this.renderGraph();
                    }
                    const nodes = (this.lastGraphData && this.lastGraphData.nodes)
                        ? this.lastGraphData.nodes.get()
                        : [];
                    const add = [];
                    for (const n of nodes) {
                        const out = String((n.data && n.data.output_type) || n.label || '');
                        const pkg = String((n.data && n.data.packagePath) || '');
                        let hit = false;
                        let label = this.nodeDisplayLabel(n);
                        if (out.includes('dixinternal.Dix') || /\/dixinternal$/.test(pkg)) {
                            hit = true;
                            label = 'Dix';
                        } else if (out.includes('TimeoutProbe')) {
                            hit = true;
                            label = 'TimeoutProbe';
                        } else if (pkg.includes('/infra/diag') || out.includes('SlowRemote')) {
                            hit = true;
                            label = label || 'diag';
                        }
                        if (!hit) continue;
                        const id = String(n.id);
                        if (this.hiddenSeeds.some((s) => s.id === id)) continue;
                        add.push({ id, label });
                    }
                    // Also hide whole plugins package if present on canvas
                    const pluginPkgs = new Set();
                    for (const n of nodes) {
                        const pkg = String((n.data && n.data.packagePath) || '');
                        const parts = pkg.split('/').filter(Boolean);
                        const i = parts.lastIndexOf('plugins');
                        if (i >= 0) pluginPkgs.add(parts.slice(0, i + 1).join('/'));
                    }
                    for (const pref of pluginPkgs) {
                        const id = 'pkg:' + pref;
                        if (this.hiddenSeeds.some((s) => s.id === id)) continue;
                        add.push({ id, label: '…/plugins', packagePrefix: pref });
                    }
                    if (!add.length) {
                        this.showHideToast('当前画布没有可收敛的 diag/Dix/plugins 节点', '');
                        return;
                    }
                    this.pushHideHistory();
                    this.hiddenSeeds = [...this.hiddenSeeds, ...add];
                    this.saveHiddenSeeds();
                    this.hiddenListOpen = true;
                    this.clearHidePreview();
                    this.graphContextMenu = null;
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                    this.showHideToast(`已收敛主图（隐藏 ${add.length} 组噪声）`, add[add.length - 1].id);
                },

                hideInventoryProvider(p, event) {
                    if (event && event.stopPropagation) event.stopPropagation();
                    if (!p || !p.id) return;
                    this.hideNodeAndDownstream({
                        id: p.id,
                        label: p.label || p.function_name || p.id,
                        data: { ...p, type: 'provider', displayLabel: p.label || p.function_name },
                    });
                },

                undoLastHide() {
                    if (!(this.hideHistory && this.hideHistory.length)) return;
                    this.hiddenSeeds = this.cloneHiddenSeeds(this.hideHistory.pop());
                    this.saveHiddenSeeds();
                    if (!this.hiddenSeeds.length) this.hiddenListOpen = false;
                    this.hideToast = null;
                    if (this._hideToastTimer) {
                        clearTimeout(this._hideToastTimer);
                        this._hideToastTimer = null;
                    }
                    this.clearHidePreview();
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                },

                unhideSeed(id) {
                    const target = String(id || '');
                    this.hiddenSeeds = this.hiddenSeeds.filter((s) => s.id !== target);
                    this.saveHiddenSeeds();
                    if (!this.hiddenSeeds.length) this.hiddenListOpen = false;
                    this.clearHidePreview();
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                },

                clearHiddenSeeds() {
                    if (!this.hiddenSeeds.length) return;
                    this.pushHideHistory();
                    this.hiddenSeeds = [];
                    this.saveHiddenSeeds();
                    this.hiddenListOpen = false;
                    this.clearHidePreview();
                    this.graphContextMenu = null;
                    this.renderGraph();
                    this.fitGraphCameraSoon();
                },

                isInventoryProviderHidden(p) {
                    if (!p || !p.id) return false;
                    const id = String(p.id);
                    if (this.hiddenSeeds.some((s) => s.id === id)) return true;
                    const pkg = String(p.output_pkg || p.packagePath || '');
                    return this.hiddenSeeds.some((s) => {
                        const pref = s.packagePrefix;
                        if (!pref) return false;
                        return pkg === pref || pkg.startsWith(pref + '/');
                    });
                },

                inventoryProviderAction(p, event) {
                    if (this.isInventoryProviderHidden(p)) {
                        if (event && event.stopPropagation) event.stopPropagation();
                        this.unhideSeed(p.id);
                        return;
                    }
                    this.hideInventoryProvider(p, event);
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
