<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\View\Modal\Admin\Templates;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class TemplatesTest extends LivewireTestCase
{
    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(Templates::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->assertSet('category', '')
            ->assertSet('search', '')
            ->assertSet('showPreviewModal', false)
            ->assertSet('showCreateModal', false)
            ->assertSet('showImportModal', false)
            ->assertSet('previewSlug', null)
            ->assertSet('importError', null);
    }

    public function test_open_preview_sets_slug_and_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->call('openPreview', 'my-template')
            ->assertSet('showPreviewModal', true)
            ->assertSet('previewSlug', 'my-template');
    }

    public function test_close_preview_hides_modal_and_clears_slug(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->call('openPreview', 'my-template')
            ->call('closePreview')
            ->assertSet('showPreviewModal', false)
            ->assertSet('previewSlug', null);
    }

    public function test_open_import_modal_shows_modal_with_clean_state(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->call('openImportModal')
            ->assertSet('showImportModal', true)
            ->assertSet('importFileName', '')
            ->assertSet('importPreview', null)
            ->assertSet('importError', null);
    }

    public function test_close_import_modal_hides_modal_and_clears_state(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->call('openImportModal')
            ->call('closeImportModal')
            ->assertSet('showImportModal', false)
            ->assertSet('importError', null)
            ->assertSet('importPreview', null);
    }

    public function test_search_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->set('search', 'feature')
            ->assertSet('search', 'feature');
    }

    public function test_category_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->set('category', 'development')
            ->assertSet('category', 'development');
    }

    public function test_clear_filters_resets_search_and_category(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->set('search', 'test')
            ->set('category', 'development')
            ->call('clearFilters')
            ->assertSet('search', '')
            ->assertSet('category', '');
    }

    public function test_get_category_color_returns_correct_class_for_development(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Templates::class);

        $class = $component->instance()->getCategoryColor('development');

        $this->assertStringContainsString('blue', $class);
    }

    public function test_get_category_color_returns_correct_class_for_maintenance(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Templates::class);

        $class = $component->instance()->getCategoryColor('maintenance');

        $this->assertStringContainsString('green', $class);
    }

    public function test_get_category_color_returns_correct_class_for_custom(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Templates::class);

        $class = $component->instance()->getCategoryColor('custom');

        $this->assertStringContainsString('zinc', $class);
    }

    public function test_get_category_color_returns_default_for_unknown(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Templates::class);

        $class = $component->instance()->getCategoryColor('unknown-category');

        $this->assertNotEmpty($class);
    }

    public function test_close_create_modal_hides_modal_and_clears_state(): void
    {
        $this->actingAsHades();

        Livewire::test(Templates::class)
            ->set('showCreateModal', true)
            ->set('createTemplateSlug', 'some-template')
            ->call('closeCreateModal')
            ->assertSet('showCreateModal', false)
            ->assertSet('createTemplateSlug', null)
            ->assertSet('createVariables', []);
    }
}
