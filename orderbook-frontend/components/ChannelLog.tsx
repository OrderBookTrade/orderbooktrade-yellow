'use client';

import { useState, useEffect } from 'react';

interface LogEntry {
    id: string;
    timestamp: Date;
    type: 'session' | 'state' | 'signature' | 'error' | 'info';
    message: string;
    data?: unknown;
}

interface ChannelLogProps {
    channelId?: string;
}

// Global log store (simple solution for demo/hackathon)
const logEntries: LogEntry[] = [];
let logId = 0;

export function addLogEntry(
    type: LogEntry['type'],
    message: string,
    data?: unknown
) {
    const entry: LogEntry = {
        id: String(++logId),
        timestamp: new Date(),
        type,
        message,
        data,
    };
    logEntries.unshift(entry);

    // Keep only last 100 entries
    if (logEntries.length > 100) {
        logEntries.pop();
    }

    // Trigger re-render via custom event
    if (typeof window !== 'undefined') {
        window.dispatchEvent(new CustomEvent('channel-log-update'));
    }
}

export function ChannelLog({ channelId }: ChannelLogProps) {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [expanded, setExpanded] = useState(false);
    const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

    // Listen for log updates
    useEffect(() => {
        const handleUpdate = () => {
            setLogs([...logEntries]);
        };

        // Initial load
        handleUpdate();

        // Listen for updates
        window.addEventListener('channel-log-update', handleUpdate);
        return () => {
            window.removeEventListener('channel-log-update', handleUpdate);
        };
    }, []);

    const toggleItemExpand = (id: string) => {
        setExpandedItems(prev => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    };

    const formatTime = (date: Date) => {
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            fractionalSecondDigits: 3,
        });
    };

    const getTypeIcon = (type: LogEntry['type']) => {
        switch (type) {
            case 'session': return '🔗';
            case 'state': return '📝';
            case 'signature': return '✍️';
            case 'error': return '❌';
            case 'info': return 'ℹ️';
            default: return '•';
        }
    };

    const getTypeClass = (type: LogEntry['type']) => {
        switch (type) {
            case 'session': return 'type-session';
            case 'state': return 'type-state';
            case 'signature': return 'type-signature';
            case 'error': return 'type-error';
            default: return 'type-info';
        }
    };

    return (
        <div className={`channel-log ${expanded ? 'expanded' : ''}`}>
            <div
                className="channel-log-header"
                onClick={() => setExpanded(!expanded)}
            >
                <h3>
                    State Channel Events
                    {channelId && (
                        <span className="channel-id">
                            {channelId.slice(0, 8)}...
                        </span>
                    )}
                </h3>
                <span className="log-count">{logs.length}</span>
                <span className="expand-icon">{expanded ? '▼' : '▲'}</span>
            </div>

            {expanded && (
                <div className="channel-log-content">
                    {logs.length === 0 ? (
                        <div className="empty-state">
                            <p>No events yet</p>
                        </div>
                    ) : (
                        <div className="log-list">
                            {logs.map((entry) => (
                                <div
                                    key={entry.id}
                                    className={`log-entry ${getTypeClass(entry.type)}`}
                                >
                                    <div
                                        className="log-entry-header"
                                        onClick={() => entry.data !== undefined && toggleItemExpand(entry.id)}
                                    >
                                        <span className="log-icon">{getTypeIcon(entry.type)}</span>
                                        <span className="log-time">{formatTime(entry.timestamp)}</span>
                                        <span className="log-message">{entry.message}</span>
                                        {Boolean(entry.data !== undefined) && (
                                            <span className="expand-data">
                                                {expandedItems.has(entry.id) ? '−' : '+'}
                                            </span>
                                        )}
                                    </div>

                                    {entry.data !== undefined && expandedItems.has(entry.id) && (
                                        <pre className="log-data">
                                            {JSON.stringify(entry.data, null, 2)}
                                        </pre>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
