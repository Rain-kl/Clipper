import { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { ErrorInline } from '@/components/layout/error';
import { EmptyStateWithBorder } from '@/components/layout/empty';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import { Layers, ListRestart, LucideIcon, Trash2 } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area';
import { useTranslations } from 'next-intl';

interface ManagePageProps<T> {
  title: string;
  icon?: LucideIcon;
  data: T[];
  loading: boolean;
  error: Error | null;
  onReload: () => void;

  /** 获取初始编辑数据 */
  getInitialEditData: (item: T) => Partial<T>;
  onSave: (item: T, editData: Partial<T>) => Promise<void>;
  onDelete?: (item: T) => Promise<void>;

  /** 渲染表格 (Config-Driven) */
  columns: {
    header: string;
    cell: (item: T) => React.ReactNode;
    width?: string;
    align?: 'left' | 'center' | 'right';
    className?: string;
  }[];

  /** 渲染详情 */
  renderDetail: (props: {
    selected: T | null;
    hovered: T | null;
    editData: Partial<T>;
    onEditDataChange: (field: keyof T, value: T[keyof T]) => void;
    onSave: () => void;
    saving: boolean;
  }) => React.ReactNode;

  /** 空状态图标 */
  emptyIcon?: LucideIcon;
  emptyDescription?: string;
  loadingDescription?: string;
  getId: (item: T) => string | number;
  headerExtra?: React.ReactNode;
}

export function ManagePage<T>({
  title,
  icon: Icon,
  data,
  loading,
  error,
  onReload,
  getInitialEditData,
  onSave,
  onDelete,
  columns,
  renderDetail,
  emptyIcon = Layers,
  emptyDescription,
  loadingDescription,
  getId,
  headerExtra,
}: ManagePageProps<T>) {
  const t = useTranslations('general.manage');
  /** 悬停状态 */
  const [hoveredItem, setHoveredItem] = useState<T | null>(null);
  const [selectedItem, setSelectedItem] = useState<T | null>(null);
  const [editData, setEditData] = useState<Partial<T>>({});
  const [saving, setSaving] = useState(false);
  const [deletingItem, setDeletingItem] = useState<T | null>(null);

  /** 悬停处理 */
  const handleHover = (item: T | null) => {
    setHoveredItem(item);
  };

  /** 选择处理 */
  const handleSelect = (item: T) => {
    const itemId = getId(item);
    const selectedId = selectedItem ? getId(selectedItem) : null;

    if (itemId === selectedId) {
      setSelectedItem(null);
      setEditData({});
    } else {
      setSelectedItem(item);
      setEditData(getInitialEditData(item));
    }
    setHoveredItem(null);
  };

  /** 编辑数据处理 */
  const handleEditDataChange = (field: keyof T, value: T[keyof T]) => {
    setEditData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  /** 保存处理 */
  const handleSave = async () => {
    if (!selectedItem) return;

    try {
      setSaving(true);
      await onSave(selectedItem, editData);
      toast.success(t('saveSuccess'));
    } catch (error) {
      toast.error(t('saveFailed'), {
        description: error instanceof Error ? error.message : t('unknownError'),
      });
    } finally {
      setSaving(false);
    }
  };

  /** 删除处理 */
  const handleDeleteClick = (item: T) => {
    setDeletingItem(item);
  };

  const handleConfirmDelete = async () => {
    if (!deletingItem || !onDelete) return;
    try {
      await onDelete(deletingItem);
      toast.success(t('deleteSuccess'));
      setDeletingItem(null);
      // 如果删除的是当前选中的项，清除选中状态
      if (selectedItem && getId(selectedItem) === getId(deletingItem)) {
        setSelectedItem(null);
        setEditData({});
      }
    } catch (error) {
      toast.error(t('deleteFailed'), {
        description: error instanceof Error ? error.message : t('unknownError'),
      });
    }
  };

  /** 渲染内容 */
  const renderContent = () => {
    if (loading && (!data || data.length === 0)) {
      return (
        <LoadingStateWithBorder
          icon={ListRestart}
          description={loadingDescription ?? t('loading')}
        />
      );
    }

    if (error) {
      return (
        <div className='p-8 border border-dashed rounded-lg'>
          <ErrorInline
            error={error}
            onRetry={onReload}
            className='justify-center'
          />
        </div>
      );
    }

    if (!data || data.length === 0) {
      return (
        <EmptyStateWithBorder
          icon={emptyIcon}
          description={emptyDescription ?? t('noData')}
        />
      );
    }

    return (
      <ManageTable
        data={data}
        columns={columns}
        selected={selectedItem}
        hovered={hoveredItem}
        onSelect={handleSelect}
        onHover={handleHover}
        onDelete={onDelete ? handleDeleteClick : undefined}
        getId={getId}
      />
    );
  };

  return (
    <div className='py-6'>
      <div className='flex pb-2 mb-6 items-center justify-between'>
        <div className='flex items-center gap-2'>
          {Icon && <Icon className='size-5 text-primary' />}
          <div>
            <h1 className='text-2xl font-semibold tracking-tight'>{title}</h1>
          </div>
        </div>
        {headerExtra}
      </div>

      <div className='space-y-6'>
        <div>
          <div className='mb-4'>
            <div className='font-semibold'>{t('configList')}</div>
          </div>
          {renderContent()}
        </div>

        <div>
          {renderDetail({
            selected: selectedItem,
            hovered: hoveredItem,
            editData,
            onEditDataChange: handleEditDataChange,
            onSave: handleSave,
            saving,
          })}
        </div>
      </div>

      <AlertDialog
        open={!!deletingItem}
        onOpenChange={(open) => !open && setDeletingItem(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('confirmDeleteDesc')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className='bg-red-600 hover:bg-red-700'
            >
              {t('confirmDeleteAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

/** 详情面板 props */
interface ManageDetailPanelProps {
  title?: string;
  isEmpty: boolean;
  emptyDescription?: string;
  onSave?: () => void;
  saving?: boolean;
  children: React.ReactNode;
}

/** 详情面板 */
export function ManageDetailPanel({
  title,
  isEmpty,
  emptyDescription,
  onSave,
  saving = false,
  children,
}: ManageDetailPanelProps) {
  const t = useTranslations('general.manage');
  const displayTitle = title ?? t('configInfo');
  const displayEmptyDescription = emptyDescription ?? t('selectConfigToView');
  if (isEmpty) {
    return (
      <div className='space-y-4'>
        <div className='font-semibold mb-4'>{displayTitle}</div>
        <EmptyStateWithBorder
          icon={Layers}
          description={displayEmptyDescription}
        />
      </div>
    );
  }

  return (
    <div className='space-y-4 sticky top-6'>
      <div>
        <div className='flex items-center justify-between mb-4'>
          <div className='font-semibold'>{displayTitle}</div>
          {onSave && (
            <Button
              onClick={onSave}
              disabled={saving}
              size='sm'
              className='px-3 h-7 text-xs'
            >
              {saving ? (
                <>
                  <Spinner /> {t('updating')}
                </>
              ) : (
                t('update')
              )}
            </Button>
          )}
        </div>
        {children}
      </div>
    </div>
  );
}

/** 表格面板 */
export function ManageTable<T>({
  data,
  columns,
  selected,
  hovered,
  onSelect,
  onHover,
  onDelete,
  getId,
}: {
  data: T[];
  columns: {
    header: string;
    cell: (item: T) => React.ReactNode;
    width?: string;
    align?: 'left' | 'center' | 'right';
    className?: string;
  }[];
  selected: T | null;
  hovered: T | null;
  onSelect: (item: T) => void;
  onHover: (item: T | null) => void;
  onDelete?: (item: T) => void;
  getId: (item: T) => string | number;
}) {
  const t = useTranslations('general.manage');
  return (
    <div className='border border-dashed shadow-none rounded-lg overflow-hidden'>
      <ScrollArea className='w-full'>
        <div className='relative w-full'>
          <table className='w-full caption-bottom text-sm'>
            <TableHeader>
              <TableRow className='border-b border-dashed'>
                {columns.map((col, index) => (
                  <TableHead
                    key={index}
                    className={`whitespace-nowrap ${col.width ? `w-[${col.width}]` : ''} ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''} ${col.className || ''}`}
                  >
                    {col.header}
                  </TableHead>
                ))}
                {onDelete && (
                  <TableHead className='whitespace-nowrap text-center w-[120px]'>
                    {t('actions')}
                  </TableHead>
                )}
              </TableRow>
            </TableHeader>
            <TableBody className='animate-in fade-in duration-200'>
              {data.map((item) => {
                const id = getId(item);
                const isSelected = selected && getId(selected) === id;
                const isHovered = hovered && getId(hovered) === id;

                return (
                  <TableRow
                    key={id}
                    className={`border-b border-dashed cursor-pointer transition-colors ${
                      isSelected
                        ? 'bg-primary/5 hover:bg-primary/10'
                        : isHovered
                          ? 'bg-gray-100 hover:bg-muted/50'
                          : 'hover:bg-gray-100'
                    }`}
                    onMouseEnter={() => onHover(item)}
                    onMouseLeave={() => onHover(null)}
                    onClick={() => onSelect(item)}
                  >
                    {columns.map((col, index) => (
                      <TableCell
                        key={index}
                        className={`text-xs py-1 ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''} ${col.className || ''}`}
                      >
                        {col.cell(item)}
                      </TableCell>
                    ))}
                    {onDelete && (
                      <TableCell className='text-xs py-1 text-center'>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='h-6 w-6 text-muted-foreground hover:text-red-600 hover:bg-red-50'
                          onClick={(e) => {
                            e.stopPropagation();
                            onDelete(item);
                          }}
                        >
                          <Trash2 className='h-3.5 w-3.5' />
                        </Button>
                      </TableCell>
                    )}
                  </TableRow>
                );
              })}
            </TableBody>
          </table>
        </div>
        <ScrollBar orientation='horizontal' />
      </ScrollArea>
    </div>
  );
}
