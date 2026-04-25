<label class="inline-flex items-center gap-3 text-sm text-zinc-900">
    <input
        id="{{ $id }}"
        type="checkbox"
        role="switch"
        @disabled($disabled)
        {{ $attributes->merge([
            'class' => 'h-5 w-9 rounded-full border-zinc-300 text-violet-600 focus:ring-violet-500 disabled:bg-zinc-100',
        ]) }}
    />

    @if($label)
        <span>{{ $label }}</span>
    @endif
</label>
