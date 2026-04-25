<label class="block space-y-2">
    @if($label)
        <span class="text-sm font-medium text-zinc-900">{{ $label }}@if($required) * @endif</span>
    @endif

    <textarea
        id="{{ $id }}"
        rows="{{ $rows }}"
        placeholder="{{ $placeholder }}"
        @disabled($disabled)
        @required($required)
        {{ $attributes->merge([
            'class' => 'w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 shadow-sm focus:border-violet-500 focus:outline-none focus:ring-2 focus:ring-violet-200 disabled:bg-zinc-100 disabled:text-zinc-500',
        ]) }}
    >{{ $slot }}</textarea>

    @if($hint)
        <span class="text-xs text-zinc-500">{{ $hint }}</span>
    @endif
</label>
