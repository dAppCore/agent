<label class="inline-flex items-center gap-3 text-sm text-zinc-900">
    <input
        id="{{ $id }}"
        type="checkbox"
        @disabled($disabled)
        @required($required)
        {{ $attributes->merge([
            'class' => 'h-4 w-4 rounded border-zinc-300 text-violet-600 focus:ring-violet-500 disabled:bg-zinc-100',
        ]) }}
    />

    @if($label)
        <span>{{ $label }}</span>
    @endif
</label>
